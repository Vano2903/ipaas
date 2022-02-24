package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

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
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
	c            *Controller
)

type Controller struct {
	ctx context.Context
	cli *client.Client
}

func NewController() (*Controller, error) {
	var err error

	c := new(Controller)
	c.ctx = context.Background()

	c.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c Controller) CreateNewDB(image, port string, env []string) (string, error) {
	//pull the image, it wont pull it if it is already there
	//it will update itself since if there is no tag by default it means latest
	out, err := c.cli.ImagePull(c.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer out.Close()
	//!could be used to sent to the client the output of the pull
	// io.Copy(os.Stdout, out)

	//!container configurations

	//container config
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
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 5,
		},
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

func (c Controller) GetContainerExternalPort(id, containerPort string) (string, error) {
	container, err := c.cli.ContainerInspect(c.ctx, id)
	if err != nil {
		return "", err
	}
	return container.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%s/tcp", containerPort))][0].HostPort, nil
}

func (c *Controller) FindVolume(name string) (volume *types.Volume, err error) {
	volumes, err := c.cli.VolumeList(context.Background(), filters.NewArgs())
	if err != nil {
		return nil, err
	}

	for _, v := range volumes.Volumes {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, nil
}

func (c *Controller) EnsureVolume(name string) (created bool, volume *types.Volume, err error) {
	volume, err = c.FindVolume(name)
	if err != nil {
		return false, nil, err
	}

	if volume != nil {
		return false, volume, nil
	}

	vol, err := c.cli.VolumeCreate(c.ctx, volumeType.VolumeCreateBody{
		Driver: "local",
		Labels: map[string]string{"matricola": "18008", "type": "db", "dbType": "mysql"},
		Name:   name,
	})
	return true, &vol, err
}

func (c *Controller) RemoveVolume(name string) (removed bool, err error) {
	vol, err := c.FindVolume(name)
	if err != nil {
		return false, err
	}

	if vol == nil {
		return false, nil
	}

	err = c.cli.VolumeRemove(context.Background(), name, true)
	if err != nil {
		return false, err
	}

	return true, nil
}

//function to generate a random alphanumerical password without spaces (24 characters)
func generatePassword() string {
	minNum := 4
	minUpperCase := 4
	passwordLength := 16

	rand.Seed(time.Now().Unix())
	var password strings.Builder

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
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

//testing
