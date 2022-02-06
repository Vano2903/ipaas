package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var (
	MYSQL_IMAGE = "mysql"
	MYSQL_PORT  = "3306"
)

func CreateNewDB(image, port string, env []string) (string, error) {
	//set the context and client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}

	//pull the image, it wont pull it if it is already there
	//it will update itself since if there is no tag by default it means latest
	out, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
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
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	//start the container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

//function to generate a random alphanumerical password without spaces
func GenerateRandomPassword() string {
	var password string
	for i := 0; i < 16; i++ {
		password += string(rand.Intn(26) + 65)
	}
	return password
}

func main() {
	fmt.Println("password:", GenerateRandomPassword())

	// env := []string{
	// 	"MYSQL_ROOT_PASSWORD=root",
	// 	"MYSQL_DATABASE=test",
	// }

	// fmt.Println(CreateNewDB(MYSQL_IMAGE, MYSQL_PORT, env))
}
