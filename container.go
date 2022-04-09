package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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
	fmt.Println("container port", containerPort)
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

	//create the container
	//!set a name to identify the container (<student-name>.<registration_number>-<db-name>)
	resp, err := c.cli.ContainerCreate(c.ctx, config, hostConfig, nil, nil, "")
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
	time.Sleep(time.Second)

	//same as docker inspect <id>
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	//from the network settings we get the port that the container is
	//listening to internally and from there we get the host one
	i := 0
	var natted []nat.PortBinding
	for {
		time.Sleep(time.Second)
		if i > 5 {
			return "", fmt.Errorf("error getting the port of the container")
		}
		i++
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