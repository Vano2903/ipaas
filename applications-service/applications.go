package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/vano2903/ipaas/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AppPost struct {
	GithubRepoUrl string `json:"github-repo,omitempty"`
	GithubBranch  string `json:"github-branch,omitempty"`
	Language      string `json:"language,omitempty"`
	Port          string `json:"port,omitempty"`
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
	GithubRepo     string             `bson:"githubRepo,omitempty" json:"githubRepo,omitempty"`
	GithubBranch   string             `bson:"githubBranch,omitempty" json:"githubBranch,omitempty"`
	LastCommitHash string             `bson:"lastCommitHash,omitempty" json:"lastCommitHash,omitempty"`
	Port           int                `bson:"port" json:"port,omitempty"`
	ExternalPort   string             `bson:"externalPort" json:"externalPort,omitempty"`
	Lang           string             `bson:"lang" json:"lang,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt,omitempty"`
	LastUpdate     time.Time          `bson:"lastUpdate" json:"lastUpdate,omitempty"`
	IsPublic       bool               `bson:"isPublic" json:"isPublic"`
	IsUpdatable    bool               `bson:"isUpdatable,omitempty" json:"isUpdatable"`
	ImgID          string             `bson:"imgID,omitempty" json:"imgID,omitempty"`
	Envs           []Env              `bson:"envs,omitempty" json:"envs,omitempty"`
	Tags           []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	Stars          []string           `bson:"stars,omitempty" json:"stars,omitempty"`
}

type Env struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
}

// GetAppInfoFromContainer returns the application metadata from the container id
// if checkCommit is set to true the function will query GitHub and check if
// the last commit has changed (if the application can be updated)
func (c ContainerController) GetAppInfoFromContainer(containerId string, checkCommit bool) (Application, error) {
	db, err := u.ConnectToDB()
	if err != nil {
		return Application{}, err
	}
	defer db.Client().Disconnect(c.ctx)

	var app Application
	err = db.Collection("applications").FindOne(c.ctx, bson.M{"containerID": containerId}).Decode(&app)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Application{}, fmt.Errorf("application not found")
		} else {
			return Application{}, err
		}
	}

	if checkCommit {
		app.IsUpdatable, err = utils.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, app.GithubBranch)
		if err != nil {
			return Application{}, err
		}
	}

	return app, nil
}

// CreateNewApplication creates a new application from the studentID (creator) and the app details, it will create
// a new image and run a container on top of that image
func (c ContainerController) CreateNewApplication(studentID int, appDetails AppPost) (Application, error) {
	//connect to the db
	conn, err := u.ConnectToDB()
	if err != nil {
		return Application{}, err
	}
	defer conn.Client().Disconnect(c.ctx)

	if err := utils.ValidGithubUrl(appDetails.GithubRepoUrl); err != nil {
		return Application{}, err
	}
	logger.Debugln("Valid github url")

	//download the repo
	repo, name, hash, err := utils.DownloadGithubRepo(studentID, appDetails.GithubBranch, appDetails.GithubRepoUrl)
	if err != nil {
		return Application{}, err
	}
	logger.Debugln("Downloaded github repo")
	logger.Debugln("repo:", repo)
	logger.Debugln("name:", name)
	logger.Debugln("hash:", hash)

	//port to expose for the app
	port, err := strconv.Atoi(appDetails.Port)
	if err != nil {
		return Application{}, fmt.Errorf("error converting the port to an int: %v", err)
	}

	//create the image from the repo downloaded
	imageName, imageID, err := c.CreateImage(studentID, port, name, appDetails.GithubBranch, repo, appDetails.Language, appDetails.Envs)
	if err != nil {
		return Application{}, fmt.Errorf("error creating the image: %v", err)
	}

	logger.Debugln("name:", imageName)
	logger.Debugln("id:", imageID)

	//remove the repo after creating the application
	err = os.RemoveAll(repo)
	if err != nil {
		return Application{}, fmt.Errorf("error removing the repo: %v", err)
	}

	//create the container from the image just created
	id, err := c.CreateNewContainerFromImage(studentID, port, name, appDetails.GithubBranch, appDetails.Language, imageName)
	if err != nil {
		return Application{}, fmt.Errorf("error creating the container: %v", err)
	}

	//remove the image created
	//err = c.RemoveImage(imageID)
	//if err != nil {
	//	return Application{}, fmt.Errorf("error removing the image: %v", err)
	//}

	if err := c.cli.ContainerStart(c.ctx, id, types.ContainerStartOptions{}); err != nil {
		return Application{}, fmt.Errorf("error starting the container: %v", err)
	}

	status, err := c.GetContainerStatus(id)
	if err != nil {
		return Application{}, fmt.Errorf("error getting the status of the container: %v", err)
	}
	logger.Debugln("status:", status)

	var externalPort string
	if status == "running" {
		externalPort, err = c.GetContainerExternalPort(id, appDetails.Port)
		if err != nil {
			return Application{}, fmt.Errorf("error getting the external port: %v", err)
		}
	}
	//get the status of the application

	var app Application
	//definition information
	app.ID = primitive.NewObjectID()
	app.StudentID = studentID
	app.ContainerID = id
	app.Type = "web"
	app.CreatedAt = time.Now()

	//git information
	app.GithubRepo = appDetails.GithubRepoUrl
	app.GithubBranch = appDetails.GithubBranch
	app.LastCommitHash = hash

	//application information
	app.Port = port
	app.Lang = appDetails.Language
	app.Name = imageName
	app.Envs = appDetails.Envs

	//container information
	app.Status = status
	//app.ImgID = imageID
	app.ExternalPort = externalPort

	//social information
	app.Description = appDetails.Description

	//insert the application in the database
	_, err = conn.Collection("applications").InsertOne(c.ctx, app)
	return app, err
}

// UpdateApplication updates the application with the new information, it will kill the old container, recreate the image
// downloading the new commit from the repo and run a new container on top of the new image
// TODO: allow to update application just by changing envs
func (c ContainerController) UpdateApplication(studentID int, oldContainerID string, env []Env) (Application, error) {
	app, err := c.GetAppInfoFromContainer(oldContainerID, true)
	if err != nil {
		return Application{}, err
	}

	//check if the user is the owner and if the application is updatable
	if app.StudentID != studentID {
		return Application{}, fmt.Errorf("you are not the owner of this application")
	}

	if !app.IsUpdatable {
		return Application{}, fmt.Errorf("the application is already up to the latest commit")
	}

	repo, name, hash, err := utils.DownloadGithubRepo(studentID, app.GithubBranch, app.GithubRepo)
	if err != nil {
		return Application{}, err
	}

	newImageName, _, err := c.CreateImage(studentID, app.Port, name, app.GithubBranch, repo, app.Lang, env)
	if err != nil {
		return Application{}, err
	}

	err = c.cli.ContainerStop(c.ctx, oldContainerID, nil)
	if err != nil {
		return Application{}, err
	}

	err = c.cli.ContainerRemove(c.ctx, oldContainerID, types.ContainerRemoveOptions{})
	if err != nil {
		return Application{}, err
	}

	//create the new container
	newContainerID, err := c.CreateNewContainerFromImage(studentID, app.Port, name, app.GithubBranch, app.Lang, newImageName)
	if err != nil {
		return Application{}, err
	}

	//update the application metadata
	app.ContainerID = newContainerID
	app.LastCommitHash = hash
	app.IsUpdatable = false
	app.LastUpdate = time.Now()
	app.Envs = env

	//update the application in the db
	db, err := u.ConnectToDB()
	if err != nil {
		return Application{}, err
	}
	defer db.Client().Disconnect(c.ctx)

	_, err = db.Collection("applications").UpdateOne(c.ctx, bson.M{"containerID": oldContainerID}, bson.M{"$set": app})
	return app, err
}

// DeleteApplication deletes a container given the container id, it will check if the user owns this application
func (c ContainerController) DeleteApplication(studentID int, containerID string) error {
	//connect to the db
	conn, err := u.ConnectToDB()
	if err != nil {
		return fmt.Errorf("error connecting to the database: %v", err)
	}
	defer conn.Client().Disconnect(c.ctx)

	var app Application
	applicationCollection := conn.Collection("applications")
	err = applicationCollection.FindOne(c.ctx, bson.M{"containerID": containerID}).Decode(&app)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("there is no application with this id")
		}
		return fmt.Errorf("error getting the application from the database: %v", err)
	}

	//TODO: admins should be able to delete any application
	if app.StudentID != studentID {
		return errors.New("you don't have permission to delete this application")
	}

	//delete the container
	err = c.DeleteContainer(containerID)
	if err != nil {
		return fmt.Errorf("error deleting the container: %v", err)
	}

	_, err = applicationCollection.DeleteOne(c.ctx, bson.M{"containerID": containerID})
	return err
}

// PublishApplication makes the application public (flags isPublic to true)
func (c ContainerController) PublishApplication(studentID int, containerID string) error {
	conn, err := u.ConnectToDB()
	if err != nil {
		return err
	}
	defer conn.Client().Disconnect(c.ctx)

	//get the application from the database
	_, err = conn.Collection("applications").UpdateOne(
		c.ctx,
		bson.M{"containerID": containerID, "studentID": studentID},
		bson.M{"$set": bson.M{"isPublic": true}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("no application found, check if the container id is correct or make sure you own this container")
		}
		return fmt.Errorf("error getting the application: %v", err)
	}
	return nil
}

// UnpublishApplication makes the application private (flags isPublic to false)
func (c ContainerController) UnpublishApplication(studentID int, containerID string) error {
	conn, err := u.ConnectToDB()
	if err != nil {
		return err
	}
	defer conn.Client().Disconnect(c.ctx)

	//get the application from the database
	_, err = conn.Collection("applications").UpdateOne(
		c.ctx,
		bson.M{"containerID": containerID, "studentID": studentID},
		bson.M{"$set": bson.M{"isPublic": false}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("no application found, check if the container id is correct or make sure you own this container")
		}
		return fmt.Errorf("error getting the application: %v", err)
	}
	return nil
}
