package main

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type dbContainerConfig struct {
	name  string
	image string
	port  string
}

type dbPost struct {
	DbDescription     string `json:"dbDescription,omitemtpy"`
	DbName            string `json:"databaseName"`
	DbType            string `json:"databaseType"`
	DbVersion         string `json:"databaseVersion"`
	DbTableCollection string `json:"databaseTable"`
}

// TODO: ADD DB NAME
// create a new database container given the db type, image, port, enviroment variables and volume
// it returns the container id and an error
func (c ContainerController) CreateNewDB(conf dbContainerConfig, env []string) (string, error) {
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

	// networkConf := &network.NetworkingConfig{
	// 	EndpointsConfig: map[string]*network.EndpointSettings{
	// 		"ipaas-network": {
	// 			NetworkID: "cf8bc28f9dcd413ece744e799e24c9be15cecae17e8225dc7e3b25db97644e10",
	// 		},
	// 	},
	// }

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
