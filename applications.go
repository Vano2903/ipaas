package main

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type AppPost struct {
	GithubRepoUrl string `json:"github-repo"`
	GithubBranch  string `json:"github-branch"`
	Language      string `json:"language"`
	Port          string `json:"port"`
	Description   string `json:"description,omitempty"`
}

type Application struct {
	ID             int       `json:"id"`
	ContainerID    string    `json:"containerID"`
	Status         string    `json:"status"`
	StudentID      int       `json:"studentID"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	GithubRepo     string    `json:"githubRepo"`
	LastCommitHash string    `json:"lastCommitHash"`
	CreatedAt      time.Time `json:"createdAt"`
	IsPublic       bool      `json:"isPublic"`
}

//create a container from an image which is the one created from a student's repository
func (c ContainerController) CreateNewApplicationFromRepo(creatorID int, port, name, language, imageName string) (string, error) {
	//generic configs for the container
	containerConfig := &container.Config{
		Image: imageName,
	}

	// externalPort, err := getFreePort()
	// if err != nil {
	// 	return "", err
	// }

	//host bindings config, hostPort is not set cause the engine will assign a dinamyc one
	hostBinding := nat.PortBinding{
		HostIP: "0.0.0.0",
		//HostPort is the port that the host will listen to, since it's not set
		//the docker engine will assign a random open port
		// HostPort: strconv.Itoa(externalPort),
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
			MaximumRetryCount: 3,
		},
	}

	//create the container
	containerBody, err := c.cli.ContainerCreate(c.ctx, containerConfig,
		hostConfig, nil, nil, fmt.Sprintf("%d-%s-%s", creatorID, name, language))
	if err != nil {
		return "", err
	}

	if err := c.cli.ContainerStart(c.ctx, containerBody.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return containerBody.ID, nil
}
