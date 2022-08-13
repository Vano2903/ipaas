package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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

//create a new controller
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

//given the creator id, port to expose (in the docker), name of the app, path for the tmp file, lang for the dockerfile and envs
//it will return the image name, image id and a possible error
func (c ContainerController) CreateImage(creatorID, port int, name, path, language string, envs []Env) (string, string, error) {
	//check if the language is supported
	var found bool
	var lang LangsStruct
	for _, l := range Langs {
		if l.Lang == language {
			found = true
			lang = l
			break
		}
	}

	//check if it's found
	if !found {
		var validLanguages string
		for _, l := range Langs {
			validLanguages += l.Lang + ", "
		}
		return "", "", fmt.Errorf("language %s not supported, the supported langs are: %v", language, validLanguages)
	}

	//get the dockerfile
	dockerfile, err := ioutil.ReadFile(fmt.Sprintf("dockerfiles/%s", lang.Dockerfile))
	if err != nil {
		return "", "", err
	}

	//set the env variables in a string with syntax
	//ENV key value
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
	f.WriteString(dockerfileWithEnvs)
	f.Close()

	fmt.Println("created the dockerfile")

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

	//read the resp.Body, its a way to wait for the image to be created
	a, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", "", err
	}
	fmt.Println("body:", string(a))

	imageCompiledCorrectly, err := c.CheckIfImageCompiled(imageName[0], string(a))
	if err != nil {
		return "", "", err
	}

	if !imageCompiledCorrectly {
		return "", "", fmt.Errorf("image %s compiled incorrectly", imageName[0])
	}

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

func (c ContainerController) CheckIfImageCompiled(imageName string, imageBuildOutput string) (bool, error) {
	fmt.Println("image build output:", imageBuildOutput)
	lines := strings.Split(imageBuildOutput, "\n")
	fmt.Println("len lines:", len(lines))
	fmt.Println("lines:", lines[len(lines)-2])
	fmt.Println("lines:", lines[len(lines)-1])
	return true, nil
}

//remove an image from the image id
func (c ContainerController) RemoveImage(imageID string) error {
	//remove the image
	_, err := c.cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force: true,
	})
	return err
}

//get the first port opened by the container on the host machine,
//the sleeps are for windows, when tested on linux they were not necesseary
func (c ContainerController) GetContainerExternalPort(id, containerPort string) (string, error) {
	//!the sleeps looks like it's only required for windows
	// time.Sleep(time.Second)

	//same as docker inspect <id>
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	//from the network settings we get the port that the container is
	//listening to internally and from there we get the host one
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

//search a volume by name and returns a pointer to the volume (type volumeType.Volume) and an error
//if the volume doesn't exist the volume pointer will be nil
func (c *ContainerController) FindVolume(name string) (volume *types.Volume, err error) {
	//get all the volumes
	volumes, err := c.cli.VolumeList(context.Background(), filters.NewArgs())
	if err != nil {
		return nil, err
	}

	//search the volume with the same name
	for _, v := range volumes.Volumes {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, nil
}

//check if a volume exists, if so returns false, the volume and an error
//if it doesn't exists it will be created and the output will be true, the volume and an error
func (c *ContainerController) EnsureVolume(name string) (created bool, volume *types.Volume, err error) {
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
		Labels: map[string]string{"matricola": "18008", "type": "db", "dbType": "mysql"},
		Name:   name,
	})
	return true, &vol, err
}

//delete a volume (only if the volume exists, if it doesnt the function will return false)
func (c *ContainerController) RemoveVolume(name string) (removed bool, err error) {
	//search the volume
	vol, err := c.FindVolume(name)
	if err != nil {
		return false, err
	}

	if vol == nil {
		return false, nil
	}

	//remove the volume
	err = c.cli.VolumeRemove(context.Background(), name, true)
	if err != nil {
		return false, err
	}

	return true, nil
}

//forcefully remove a container from the container id
func (c *ContainerController) DeleteContainer(containerID string) error {
	return c.cli.ContainerRemove(c.ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

func (c *ContainerController) GetContainerLogs(containerID string) (string, error) {
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
	logs, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

func (c *ContainerController) GetContainerStatus(id string) (string, error) {
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	return container.State.Status, nil
}
