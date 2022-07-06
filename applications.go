package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AppPost struct {
	GithubRepoUrl string `json:"github-repo"`
	GithubBranch  string `json:"github-branch"`
	Language      string `json:"language"`
	Port          string `json:"port"`
	Description   string `json:"description,omitempty"`
	Envs          []Env  `json:"envs,omitempty"`
}

type Application struct {
	ID             primitive.ObjectID `bson:"_id" json:"-"`
	ContainerID    string             `bson:"containerID" json:"containerID,omitempty"`
	Status         string             `bson:"status" json:"status,omitempty"`
	StudentID      int                `bson:"studentID" json:"studentID,omitempty"`
	Type           string             `bson:"type" json:"type,omitempty"`
	Name           string             `bson:"name" json:"name,omitempty"`
	Description    string             `bson:"description" json:"description,omitempty"`
	GithubRepo     string             `bson:"githubRepo,omitemtpy" json:"githubRepo,omitempty"`
	GithubBranch   string             `bson:"githubBranch,omitemtpy" json:"githubBranch,omitempty"`
	LastCommitHash string             `bson:"lastCommitHash,omitemtpy" json:"lastCommitHash,omitempty"`
	Port           string             `bson:"port" json:"port,omitempty"`
	ExternalPort   string             `bson:"externalPort" json:"externalPort,omitempty"`
	Lang           string             `bson:"lang" json:"lang,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt,omitempty"`
	IsPublic       bool               `bson:"isPublic" json:"isPublic"`
	IsUpdatable    bool               `bson:"isUpdatable,omitempty" json:"isUpdatable"`
	Img            string             `bson:"img,omitempty" json:"img,omitempty"`
	Envs           []Env              `bson:"envs,omitempty" json:"envs,omitempty"`
	Tags           []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	Stars          []string           `bson:"stars,omitempty" json:"stars,omitempty"`
}

type Env struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
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
func (c ContainerController) GetAppInfoFromContainer(containerId string, checkCommit bool, util *Util) (Application, error) {
	db, err := connectToDB()
	if err != nil {
		return Application{}, err
	}
	defer db.Client().Disconnect(context.TODO())

	var app Application
	err = db.Collection("applications").FindOne(context.TODO(), bson.M{"containerID": containerId}).Decode(&app)
	if err != nil {
		return Application{}, err
	}

	if checkCommit {
		app.IsUpdatable, err = util.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, app.GithubBranch)
		if err != nil {
			return Application{}, err
		}
	}

	// if getEnvs {
	// 	envs := make(map[string]string)
	// 	results, err := db.Query("SELECT * FROM envs WHERE applicationID=?", app.ID)
	// 	if err != nil {
	// 		return Application{}, nil, err
	// 	}
	// 	for results.Next() {
	// 		var key, value string
	// 		err = results.Scan(&key, &value)
	// 		if err != nil {
	// 			return Application{}, nil, err
	// 		}
	// 		envs[key] = value
	// 	}
	// 	return app, envs, nil
	// }

	return app, nil
}
