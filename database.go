package main

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	//constants for mysql
	MYSQL_IMAGE = "mysql"
	MYSQL_PORT  = "3306"
)

var (
	c           *Controller
	currentPath string
)

type Controller struct {
	ctx context.Context //context for the docker client
	cli *client.Client  //docker client
}

//create a new controller
func NewController() (*Controller, error) {
	var err error

	c := new(Controller)
	c.ctx = context.Background()

	//creating docker client from env
	c.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return c, nil
}

//create a new database container given the db type, image, port, enviroment variables and volume
//it returns the container id and an error
func (c Controller) CreateNewDB(DbType, image, port string, env []string) (string, error) {
	//pull the image, it wont pull it if it is already there
	//it will update itself since if there is no tag by default it means latest
	out, err := c.cli.ImagePull(c.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer out.Close()
	//!could be used to sent to the client the output of the pull
	// io.Copy(os.Stdout, out)

	//container config (image and environment variables)
	config := &container.Config{
		Image: image,
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
	containerPort, err := nat.NewPort("tcp", port)
	if err != nil {
		return "", err
	}

	//set a slice of possible port bindings
	//since it's a db container we need just one
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	//set the configuration of the host
	//set the port bindings and the restart policy
	//!choose a restart policy
	//!define the volume
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

		//!should change, this is only for mysql
		// Binds: []string{
		// 	fmt.Sprintf("%s:/var/lib/mysql", volumePath),
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

	return resp.ID, nil
}

//get the first port opened by the container on the host machine
func (c Controller) GetContainerExternalPort(id, containerPort string) (string, error) {
	//same as docker inspect <id>
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	//from the network settings we get the port that the container is
	//listening to internally and from there we get the host one
	return container.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%s/tcp", containerPort))][0].HostPort, nil
}

//search a volume by name and returns a pointer to the volume (type volumeType.Volume) and an error
//if the volume doesn't exist the volume pointer will be nil
func (c *Controller) FindVolume(name string) (volume *types.Volume, err error) {
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
func (c *Controller) EnsureVolume(name string) (created bool, volume *types.Volume, err error) {
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
func (c *Controller) RemoveVolume(name string) (removed bool, err error) {
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

func init() {
	var err error
	c, err = NewController()
	if err != nil {
		log.Fatal(err)
	}
}

// func main() {
// password := generatePassword()
// fmt.Println("password:", password)

// env := []string{
// 	"MYSQL_ROOT_PASSWORD=" + password,
// 	"MYSQL_USER=test",
// 	"MYSQL_PASSWORD=test",
// 	"MYSQL_DATABASE=test",
// }

// fmt.Println(c.EnsureVolume("18008-mysql"))
// fmt.Println(c.FindVolume("18008-mysql"))

// fmt.Println("creating new db")
// id, err := c.CreateNewDB(MYSQL_IMAGE, MYSQL_PORT, env)
// fmt.Println(id, err)

// username := "test"
// pass := "test"
// dbName := "test"
// port, _ := c.GetContainerExternalPort(id, MYSQL_PORT)
// fmt.Println("username", username)
// fmt.Println("password", pass)
// fmt.Println("database", dbName)
// fmt.Println("port", port)
// fmt.Println("uri", fmt.Sprintf("%s:%s@tcp(localhost:%s)/%s", username, pass, port, dbName))
// fmt.Println("aaa")
// }

// func main() {
// 	password := generatePassword()
// 	fmt.Println("password:", password)

// 	env := []string{
// 		"MYSQL_ROOT_PASSWORD=" + password,
// 		"MYSQL_USER=test",
// 		"MYSQL_PASSWORD=test",
// 		"MYSQL_DATABASE=test",
// 	}

// 	fmt.Println("creating new db")
// 	DbType := "mysql"
// 	id, err := c.CreateNewDB(DbType, MYSQL_IMAGE, MYSQL_PORT, env, fmt.Sprintf("%s/%s", currentPath, "testvolume")) //currentPath+"/testvolume"
// 	fmt.Println(id, err)

// 	username := "test"
// 	pass := "test"
// 	dbName := "test"
// 	port, _ := c.GetContainerExternalPort(id, MYSQL_PORT)
// 	fmt.Println("username", username)
// 	fmt.Println("password", pass)
// 	fmt.Println("database", dbName)
// 	fmt.Println("port", port)
// 	fmt.Println("uri", fmt.Sprintf("%s:%s@tcp(localhost:%s)/%s", username, pass, port, dbName))
// }
