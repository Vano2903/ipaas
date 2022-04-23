package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
)

type dbPost struct {
	DbName            string `json:"databaseName"`
	DbType            string `json:"databaseType"`
	DbVersion         string `json:"databaseVersion"`
	DbTableCollection string `json:"databaseTable"`
}

type ContainerController struct {
	ctx                 context.Context //context for the docker client
	cli                 *client.Client  //docker client
	dbContainersConfigs map[string]dbContainerConfig
}

type dbContainerConfig struct {
	name  string
	image string
	port  string
}

func (c ContainerController) CreateNewApplicationFromRepo(creatorID int, port, name, path, language string, envs [][]string) (string, error) {
	//check if the language is supported
	var found bool
	for _, l := range Langs {
		if l == language {
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("language %s not supported, the supported langs are: %v", language, Langs)
	}

	//get the dockerfile
	dockerfile, err := ioutil.ReadFile(fmt.Sprintf("dockerfiles/%s.dockerfile", language))
	if err != nil {
		return "", err
	}

	//set the env variables in a string with syntax
	//ENV key value
	var envString string
	if envs != nil {
		for _, e := range envs {
			if len(e) != 2 {
				return "", fmt.Errorf("invalid env %v, the len of the environment must be 2", e)
			}
			envString += fmt.Sprintf("ENV %s %s ", e[0], e[1])
		}
	}

	//get a random open port
	// randomPort, err := getFreePort()
	// if err != nil {
	// 	return "", err
	// }

	//create the dockerfile
	dockerfileWithEnvs := fmt.Sprintf(string(dockerfile), name, path, envString)
	//set a random name for the dockerfile
	dockerName := "ipaas-dockerfile_" + generateRandomString(10)

	//create and write the propretary dockerfile to the repo
	f, err := os.Create(path + "/" + dockerName)
	if err != nil {
		panic(err)
	}
	f.WriteString(dockerfileWithEnvs)
	f.Close()

	//create a build context, is a tar with the temp repo,
	//needed since we are not using the filesystem as a context
	buildContext, err := archive.TarWithOptions(path, &archive.TarOptions{
		NoLchown: true,
	})
	defer buildContext.Close()
	if err != nil {
		return "", err
	}

	//create the image from the dockerfile
	//we are setting some default labels and the flag -rm -f
	//!should set memory and cpu limit
	resp, err := c.cli.ImageBuild(c.ctx, buildContext, types.ImageBuildOptions{
		Dockerfile: dockerName,
		//Squash: true,
		Tags: []string{fmt.Sprintf("%d-%s-%s", creatorID, name, language)},
		Labels: map[string]string{
			"creator": fmt.Sprintf("%d", creatorID),
			"lang":    language,
			"name":    name,
		},
		// Remove:      true,
		// ForceRemove: true,
	})
	if err != nil {
		return "", err
	}

	//read the resp.Body to get the id of the image
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("body:", string(body))
	// return "", nil

	hostBinding := nat.PortBinding{
		HostIP: "0.0.0.0",
		//HostPort is the port that the host will listen to, since it's not set
		//the docker engine will assign a random open port
		// HostPort: "8080",
	}

	//set the port for the container (internal one)
	containerPort, err := nat.NewPort("tcp", port)
	fmt.Println("container port" + containerPort)
	if err != nil {
		return "", err
	}

	//set a slice of possible port bindings
	//since it's a db container we need just one
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	//set the configuration of the host
	//set the port bindings and the restart policy
	//!choose a restart policy
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3,
		},
		// Mounts: []mount.Mount{
		// 	{
		// 		Type:   "volume",
		// 		Source: volumePath,
		// 		Target: "/var/lib/mysql",
		// 	},
		// },
	}

	//create the container
	containerBody, err := c.cli.ContainerCreate(c.ctx, &container.Config{
		Image: fmt.Sprintf("%d-%s-%s", creatorID, name, language),
	}, hostConfig, nil, nil, fmt.Sprintf("%d-%s-%s", creatorID, name, language))
	if err != nil {
		return "", err
	}
	fmt.Println("container id:", containerBody.ID)

	if err := c.cli.ContainerStart(c.ctx, containerBody.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	fmt.Println("started")

	return containerBody.ID, nil
}

//create a new database container given the db type, image, port, enviroment variables and volume
//it returns the container id and an error
func (c ContainerController) CreateNewDB(conf dbContainerConfig, env []string) (string, error) {
	// //pull the image, it wont pull it if it is already there
	// //it will update itself since if there is no tag by default it means latest
	// out, err := c.cli.ImagePull(c.ctx, conf.image, types.ImagePullOptions{})
	// if err != nil {
	// 	return "", err
	// }
	// defer out.Close()
	// //!could be used to sent to the client the output of the pull
	// // io.Copy(os.Stdout, out)

	//container config (image and environment variables)
	config := &container.Config{
		Image: conf.image,
		Env:   env,
	}
	//host config
	hostBinding := nat.PortBinding{
		HostIP: "0.0.0.0",
		//HostPort is the port that the host will listen to, since it's not set
		//the docker engine will assign a random open port
		// HostPort: "8080",
	}

	//set the port for the container (internal one)

	containerPort, err := nat.NewPort("tcp", conf.port)
	fmt.Println("container port" + containerPort)
	if err != nil {
		return "", err
	}

	//set a slice of possible port bindings
	//since it's a db container we need just one
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	//set the configuration of the host
	//set the port bindings and the restart policy
	//!choose a restart policy
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 5,
		},
		// Mounts: []mount.Mount{
		// 	{
		// 		Type:   "volume",
		// 		Source: volumePath,
		// 		Target: "/var/lib/mysql",
		// 	},
		// },
	}

	networkConf := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"ipaas-network": {
				NetworkID: "cf8bc28f9dcd413ece744e799e24c9be15cecae17e8225dc7e3b25db97644e10",
			},
		},
	}

	//create the container
	//!set a name to identify the container (<student-name>.<registration_number>-<db-name>)
	resp, err := c.cli.ContainerCreate(c.ctx, config, hostConfig, networkConf, nil, "")
	if err != nil {
		return "", err
	}

	//start the container
	if err := c.cli.ContainerStart(c.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	// statusCh, errCh := c.cli.ContainerWait(c.ctx, resp.ID, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		return "", err
	// 	}
	// case <-statusCh:
	// }

	return resp.ID, nil
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

//create a new controller
func NewContainerController() (*ContainerController, error) {
	var err error

	c := new(ContainerController)
	c.ctx = context.Background()

	//creating docker client from env
	c.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	c.dbContainersConfigs = map[string]dbContainerConfig{
		"mysql": {
			name:  "mysql",
			image: "mysql:8.0.28-oracle",
			port:  "3306",
		},
		"mariadb": {
			name:  "mariadb",
			image: "mariadb:10.8.2-rc-focal",
			port:  "3306",
		},
		"mongodb": {
			name:  "mongodb",
			image: "mongo:5.0.6",
			port:  "27017",
		},
	}

	return c, nil
}

//!BACKLOG, UNDER DEVELOPMENT

// func (c ContainerController) ExportDBData(containerID string, dbData dbPost) (string, error) {
// 	//docker exec CONTAINER /usr/bin/mysqldump -u root --password=root DATABASE > /tmp/DATABASE.sql

// 	//check if the container id is an exadecimal string
// 	_, err := strconv.ParseUint(containerID, 16, 64)
// 	if err != nil {
// 		return "", errors.New("not a valid container id")
// 	}

// 	//get the container
// 	container, err := c.cli.ContainerInspect(c.ctx, containerID)
// 	if err != nil {
// 		return "", err
// 	}

// 	//get the database password

// 	switch dbData.DbType {
// 	case "mysql", "mariadb":
// 		//get the database password from enviroment variables
// 		dbPassword := container.Config.Env[1]
// 		//create the command
// 		cmd := fmt.Sprintf("docker exec %s /usr/bin/mysqldump -u root --password=%s %s > /tmp/%s.sql", dbPassword, dbName, containerID)
// 		//execute the command
// 		out, err := exec.Command("sh", "-c", cmd).Output()
// 		if err != nil {
// 			return "", err
// 		}
// 		log.Println("output", out)
// 	case "mongodb":
// 		//get the database password from enviroment variables
// 		dbPassword := container.Config.Env[2]
// 		//create the command
// 		cmd := fmt.Sprintf("docker exec %s /usr/bin/mysqldump -u root --password=%s %s > /tmp/%s.sql", dbPassword, dbName, containerID)
// 		//execute the command
// 		out, err := exec.Command("sh", "-c", cmd).Output()
// 		if err != nil {
// 			return "", err
// 		}
// 		log.Println("output", out)

// 	}
// }
