package main

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type AppPost struct {
	GithubRepoUrl string            `json:"github-repo"`
	GithubBranch  string            `json:"github-branch"`
	Language      string            `json:"language"`
	Port          string            `json:"port"`
	Description   string            `json:"description,omitempty"`
	Envs          map[string]string `json:"envs,omitempty"`
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
	GithubBranch   string    `json:"githubBranch"`
	LastCommitHash string    `json:"lastCommitHash"`
	Port           string    `json:"port"`
	ExternalPort   string    `json:"externalPort"`
	Lang           string    `json:"lang"`
	CreatedAt      time.Time `json:"createdAt"`
	IsPublic       bool      `json:"isPublic"`
	IsUpdatable    bool      `json:"isUpdatable"`
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

	return containerBody.ID, nil
}

//return the application metadata from the container id
//it can be specified if to check if the last commit has changed and to retrive the envs
func (c ContainerController) GetAppInfoFromContainer(containerId string, checkCommit, getEnvs bool, util *Util) (Application, map[string]string, error) {
	db, err := connectToDB()
	if err != nil {
		return Application{}, nil, err
	}
	defer db.Close()

	var app Application

	err = db.QueryRow("SELECT * FROM applications WHERE containerID = ?", containerId).Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &app.GithubRepo, &app.LastCommitHash, &app.Port, &app.ExternalPort, &app.Lang, &app.CreatedAt, &app.IsPublic)
	if err != nil {
		return Application{}, nil, err
	}

	if checkCommit {
		app.IsUpdatable, err = util.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, app.GithubBranch)
		if err != nil {
			return Application{}, nil, err
		}
	}

	if getEnvs {
		envs := make(map[string]string)
		results, err := db.Query("SELECT * FROM envs WHERE applicationID=?", app.ID)
		if err != nil {
			return Application{}, nil, err
		}
		for results.Next() {
			var key, value string
			err = results.Scan(&key, &value)
			if err != nil {
				return Application{}, nil, err
			}
			envs[key] = value
		}
		return app, envs, nil
	}

	return app, nil, nil
}
