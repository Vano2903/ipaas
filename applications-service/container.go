package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
)

type ContainerController struct {
	ctx context.Context //context for the docker client
	cli *client.Client  //docker client
}

// NewContainerController creates a new controller
func NewContainerController(ctx context.Context) (*ContainerController, error) {
	c := new(ContainerController)
	c.ctx = ctx

	//creating docker client from env
	var err error
	c.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// CreateImage will create an image given the creator id, port to expose (in the docker),
// name of the app, path for the tmp file, lang for the dockerfile and envs, if no error occurs
// the function will return the image name and image id
func (c ContainerController) CreateImage(creatorID, port int, name, path, language string, envs []Env) (string, string, error) {
	//check if the language is supported
	var found bool
	for _, l := range Langs {
		if l.Lang == language {
			found = true
			break
		}
	}

	//check if it's found
	if !found {
		return "", "", fmt.Errorf("language %s not supported, the supported langs are: %v", language, Langs)
	}

	//get the dockerfile
	dockerfile, err := os.ReadFile(fmt.Sprintf("dockerfiles/%s.dockerfile", language))
	if err != nil {
		return "", "", err
	}

	//set the env variables in a string with syntax: ENV key value
	var envString string
	for _, env := range envs {
		envString += fmt.Sprintf("ENV %s %s\n", env.Key, env.Value)
	}

	//create the dockerfile
	dockerfileWithEnvs := fmt.Sprintf(string(dockerfile), name, path, envString, port)
	//set a random name for the dockerfile
	dockerName := "ipaas-dockerfile_" + GenerateRandomString(10)

	//create and write the propretary dockerfile to the repo
	f, err := os.Create(path + "/" + dockerName)
	if err != nil {
		return "", "", err
	}
	if _, err := f.WriteString(dockerfileWithEnvs); err != nil {
		return "", "", err
	}
	if err := f.Close(); err != nil {
		return "", "", err
	}

	fmt.Println("dockerfile created")

	//create a build context, is a tar with the temp repo,
	//needed since we are not using the filesystem as a context
	buildContext, err := archive.TarWithOptions(path, &archive.TarOptions{
		NoLchown: true,
	})
	if err != nil {
		return "", "", err
	}
	defer buildContext.Close()
	fmt.Println("build the context:", &buildContext)

	//create the name for the image <creatorID>-<name>-<language>
	imageName := []string{fmt.Sprintf("%d-%s-%s", creatorID, name, language)}
	fmt.Println("image name:", imageName[0])
	//create the image from the dockerfile
	//we are setting some default labels and the flag -rm -f
	//!should set memory and cpu limit
	resp, err := c.cli.ImageBuild(c.ctx, buildContext, types.ImageBuildOptions{
		Dockerfile: dockerName,
		//Squash: true,
		Tags: imageName,
		Labels: map[string]string{
			"creator": fmt.Sprintf("%d", creatorID),
			"lang":    language,
			"name":    name,
		},
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return "", "", err
	}

	//read the resp.Body, it's a way to wait for the image to be created
	a, err := io.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		return "", "", err
	}
	if err != nil {
		return "", "", err
	}
	fmt.Println("body:", string(a))
	//find the id of the image just created
	var out bytes.Buffer

	//check if image generated errors
	// values := strings.Split(string(a), "\n")
	// for _, v := range values {}

	cmd := exec.CommandContext(c.ctx, "docker", "images", "-q", imageName[0])
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", "", err
	}

	return imageName[0], strings.Replace(out.String(), "\n", "", -1), nil
}

// RemoveImage removes an image given the image id
func (c ContainerController) RemoveImage(imageID string) error {
	//remove the image
	_, err := c.cli.ImageRemove(c.ctx, imageID, types.ImageRemoveOptions{
		Force: true,
	})
	return err
}

// GetContainerExternalPort gets the first port opened by the container on the host machine,
func (c ContainerController) GetContainerExternalPort(id, containerPort string) (string, error) {
	//same as docker inspect <id>
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}

	//reading from the network settings of the cotnainer the function gets the port
	//that the container is listening to internally and from there is able to get the host one
	//!thecnically this should only be necessary for windows but for some "good practice" we will leave it here
	i := 0
	var natted []nat.PortBinding
	for {
		time.Sleep(time.Second)
		if i > 5 {
			return "", fmt.Errorf("error getting the port of the container")
		}
		i++
		//get the external port from the docker inspect command
		natted = container.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%s/tcp", containerPort))]
		if len(natted) > 0 {
			break
		}
	}
	return natted[0].HostPort, nil
}

// FindVolume searches a volume by name and returns a pointer to the volume (type volumeType.Volume) and an error
// if the volume isn't found
func (c ContainerController) FindVolume(name string) (volume *types.Volume, err error) {
	//get all the volumes
	volumes, err := c.cli.VolumeList(c.ctx, filters.NewArgs())
	if err != nil {
		return nil, err
	}

	//search the volume with the same name
	for _, v := range volumes.Volumes {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("volume %s not found", name)
}

// EnsureVolume checks if a volume exists, if so "created" will be false,
// if it doesn't it will be created and "created" will be true
// TODO: pass labels as a parameter
func (c ContainerController) EnsureVolume(name string) (created bool, volume *types.Volume, err error) {
	//check if the volume exists (if it doesn't volume will be nil)
	volume, err = c.FindVolume(name)
	if err != nil {
		return false, nil, err
	}

	if volume != nil {
		return false, volume, nil
	}

	//create the volume given the context and the volume create body struct
	vol, err := c.cli.VolumeCreate(c.ctx, volumeType.VolumeCreateBody{
		Driver: "local",
		//Labels: map[string]string{"matricola": "18008", "type": "db", "dbType": "mysql"},
		Name: name,
	})
	return true, &vol, err
}

// RemoveVolume deletes a volume (only if the volume exists, if it doesn't the function will return false)
func (c ContainerController) RemoveVolume(name string) (removed bool, err error) {
	//search the volume
	_, err = c.FindVolume(name)
	if err != nil {
		return false, err
	}

	//remove the volume
	err = c.cli.VolumeRemove(c.ctx, name, true)
	return err == nil, err
}

// DeleteContainer forcefully removes a container from the container id
func (c ContainerController) DeleteContainer(containerID string) error {
	return c.cli.ContainerRemove(c.ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

// GetContainerLogs returns the logs of a container given the container id
func (c ContainerController) GetContainerLogs(containerID string) (string, error) {
	reader, err := c.cli.ContainerLogs(c.ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "all",
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

// GetContainerStatus returns the status of a container given the container id
func (c ContainerController) GetContainerStatus(id string) (string, error) {
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	return container.State.Status, nil
}
