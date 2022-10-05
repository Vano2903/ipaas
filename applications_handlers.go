package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
	resp "github.com/vano2903/ipaas/responser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TODO: should delete the container/image in case of an error because it would stop the user to create again the application
// new application handler let the user host a new application given:
// 1) GitHub repository
// 2) programming lang
// 3) port of the program
// ! for now the only supported applications are web based one
func (h Handler) NewApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Client().Disconnect(context.Background())

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//read post body
	var appPost AppPost
	err = json.NewDecoder(r.Body).Decode(&appPost)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error decoding the json: %v", err.Error())
		return
	}

	fmt.Println(appPost)

	//check that the appPost.GithubRepo is an actual url
	if err := h.util.ValidGithubUrl(appPost.GithubRepoUrl); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid github repo url")
		return
	}

	//download the repoPath
	repoPath, name, hash, err := h.util.DownloadGithubRepo(student.ID, appPost.GithubBranch, appPost.GithubRepoUrl)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error downloading the repo, try again in one minute: %v", err.Error())
		return
	}
	fmt.Println("repo: ", repoPath)
	fmt.Println("name: ", name)
	fmt.Println("hash: ", hash)

	//port to expose for the app
	port, err := strconv.Atoi(appPost.Port)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error converting the port to an int: %v", err.Error())
		return
	}

	//create the image from the repo downloaded
	imageName, imageID, err := h.cc.CreateImage(student.ID, port, name, appPost.GithubBranch, repoPath, appPost.Language, appPost.Envs)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error creating the image: %v", err.Error())
		return
	}

	fmt.Println("name:", imageName)
	fmt.Println("id:", imageID)

	//create the container from the image just created
	id, err := h.cc.CreateNewApplicationFromRepo(student.ID, appPost.Port, name, appPost.Language, imageName)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error creating the container: %v", err.Error())
		return
	}

	//remove the image created
	err = h.cc.RemoveImage(imageID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error removing the image: %v", err.Error())
		return
	}

	if err := h.cc.cli.ContainerStart(h.cc.ctx, id, types.ContainerStartOptions{}); err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error starting the container: %v", err.Error())
		return
	}

	//remove the repo after creating the application
	err = os.RemoveAll(repoPath)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error removing the repo: %v", err)
		return
	}

	exernalPort, err := h.cc.GetContainerExternalPort(id, appPost.Port)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the external port: %v", err.Error())
		return
	}

	//get the status of the application
	status, err := h.cc.GetContainerStatus(id)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the status of the container: %v", err.Error())
		return
	}
	fmt.Println("status:", status)

	var app Application
	app.ID = primitive.NewObjectID()
	app.ContainerID = id
	app.Status = status
	app.StudentID = student.ID
	app.Type = "web"
	app.Name = imageName
	app.Description = appPost.Description
	app.GithubRepo = appPost.GithubRepoUrl
	app.LastCommitHash = hash
	app.Port = appPost.Port
	app.Lang = appPost.Language
	app.ExternalPort = exernalPort
	app.CreatedAt = time.Now()
	app.Envs = appPost.Envs

	//insert the application in the database
	_, err = conn.Collection("applications").InsertOne(context.Background(), app)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error inserting the application in the database: %v", err.Error())
		return
	}
	toSend := map[string]interface{}{
		"container id":  id,
		"external_port": os.Getenv("IP") + ":" + exernalPort,
		"status":        status,
	}

	resp.SuccessParse(w, http.StatusOK, "application created", toSend)
}

// delete a container given the container id, it will check if the user owns this application
func (h Handler) DeleteApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the container from /{containerID}
	containerID := mux.Vars(r)["containerID"]

	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Client().Disconnect(context.Background())

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}
	applicationCollection := conn.Collection("applications")
	var app Application
	err = applicationCollection.FindOne(context.Background(), bson.M{"containerID": containerID}).Decode(&app)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Errorf(w, http.StatusBadRequest, "there is no application with this id")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application from the database: %v", err.Error())
		return
	}

	if app.StudentID != student.ID {
		resp.Errorf(w, http.StatusForbidden, "you don't have permission to delete this application")
		return
	}

	//delete the container
	err = h.cc.DeleteContainer(containerID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the container: %v", err.Error())
		return
	}

	_, err = applicationCollection.DeleteOne(context.Background(), bson.M{"containerID": containerID})
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the application: %v", err.Error())
		return
	}
	resp.Success(w, http.StatusOK, "container deleted successfully")
}

// !hold the port for the new container
// !don't use a different one
// *1) delete the old container
// *2) download the repo from GitHub
// *3) build the image
// *4) create the new container
// *5) cleanups
// ! should delete old container lastly, but I have to change the way I create images
func (h Handler) UpdateApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the container from /{containerID}
	containerID := mux.Vars(r)["containerID"]

	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Client().Disconnect(context.Background())

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//get the application from the database and check if it's owned by the student
	applicationCollection := conn.Collection("applications")
	var app Application
	err = applicationCollection.FindOne(context.Background(), bson.M{"containerID": containerID}).Decode(&app)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Errorf(w, http.StatusBadRequest, "there is no application with this id")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application from the database: %v", err.Error())
		return
	}

	if app.StudentID != student.ID {
		resp.Errorf(w, http.StatusForbidden, "you don't have permission to delete this application")
		return
	}

	//check if the commit has changed
	changed, err := h.util.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, "") //app.GithubBranch)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error checking if the commit has changed: %v", err.Error())
		return
	}

	if !changed {
		resp.Errorf(w, http.StatusBadRequest, "the repo commit has not changed, the application will not be updated")
		return
	}

	//download the repo
	//TODO: implement the branch
	repo, name, hash, err := h.util.DownloadGithubRepo(student.ID, "", app.GithubRepo)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error downloading the repo, try again in one minute: %v", err.Error())
		return
	}

	//port to expose for the app
	Intport, err := strconv.Atoi(app.Port)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error converting the port to an int: %v", err.Error())
		return
	}

	//delete the container
	err = h.cc.DeleteContainer(containerID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the container: %v", err.Error())
		return
	}

	//create the image from the repo downloaded
	imageName, _, err := h.cc.CreateImage(student.ID, Intport, name, repo, app.GithubBranch, app.Lang, app.Envs)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error creating the image: %v", err.Error())
		return
	}

	//create the container from the image just created
	newContainerID, err := h.cc.CreateNewApplicationFromRepo(student.ID, app.Port, name, app.Lang, imageName)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error creating the container: %v", err.Error())
		return
	}

	//remove the repo after creating the application
	err = os.RemoveAll(repo)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error removing the repo: %v", err)
		return
	}

	//!remove the image created
	//!not implemented for now
	// err = h.cc.RemoveImage(imageID)
	// if err != nil {
	// 	resp.Errorf(w, http.StatusInternalServerError, "error removing the image: %v", err.Error())
	// 	return
	// }

	if err := h.cc.cli.ContainerStart(h.cc.ctx, newContainerID, types.ContainerStartOptions{}); err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error starting the container: %v", err.Error())
		return
	}

	exernalPort, err := h.cc.GetContainerExternalPort(newContainerID, app.Port)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the external port: %v", err.Error())
		return
	}

	//get the status of the application
	status, err := h.cc.GetContainerStatus(newContainerID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the status of the container: %v", err.Error())
		return
	}
	fmt.Println("status:", status)

	//update the application in the database
	app.LastCommitHash = hash
	app.ExternalPort = exernalPort
	app.Status = status
	app.ContainerID = newContainerID

	_, err = applicationCollection.UpdateOne(context.Background(), bson.M{"_id": app.ID}, bson.M{"$set": app})
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error updating application: %v", err.Error())
		return
	}

	toSend := map[string]interface{}{
		"container id":  newContainerID,
		"external port": os.Getenv("IP") + ":" + exernalPort,
		"status":        status,
	}

	resp.SuccessParse(w, http.StatusOK, "application updated", toSend)
}

// it will return a json with all the applications owned by the student (even the privates one)
// this endpoint will only be accessible if logged in
func (h Handler) GetAllApplicationsOfStudentPrivate(w http.ResponseWriter, r *http.Request) {
	//get the {type} of the application from the url
	typeOfApp := mux.Vars(r)["type"]
	if typeOfApp != "database" && typeOfApp != "web" && typeOfApp != "all" && typeOfApp != "updatable" {
		resp.Error(w, http.StatusBadRequest, "Invalid type of application")
		return
	}

	onlyUpdatable := typeOfApp == "updatable"
	if onlyUpdatable {
		typeOfApp = "web"
	}

	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Client().Disconnect(context.Background())

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	applicationCollection := conn.Collection("applications")

	//get the applications from the database
	var apps []Application
	if typeOfApp == "all" {
		cur, err := applicationCollection.Find(context.TODO(), bson.M{"studentID": student.ID})
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		err = cur.All(context.TODO(), &apps)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
	} else {
		cur, err := applicationCollection.Find(context.TODO(), bson.M{"studentID": student.ID, "type": typeOfApp})
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		err = cur.All(context.TODO(), &apps)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
	}

	//parse the rows into a []Application and return it
	var applications []Application
	var errors []string
	for _, app := range apps {
		if app.GithubRepo != "" {
			app.IsUpdatable, err = h.util.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, app.GithubBranch)
			if err != nil {
				app.IsUpdatable = false
				errors = append(errors, err.Error())
			}
		}
		if onlyUpdatable {
			fmt.Printf("app %s is updatable: %v\n", app.Name, app.IsUpdatable)
			if !app.IsUpdatable {
				continue
			}
		}
		applications = append(applications, app)
	}
	if len(errors) > 0 {
		resp.SuccessParse(w, http.StatusOK, fmt.Sprintf("Applications retrived, tho some errors were found: %v", errors), applications)
		return

	}
	resp.SuccessParse(w, http.StatusOK, "Applications retrived successfully", applications)
}

// get all the public applications of a student given the student id
// you don't need to be logged in to access this endpoint
func (h Handler) GetAllApplicationsOfStudentPublic(w http.ResponseWriter, r *http.Request) {
	//get the {studentID} from the url
	studentID, err := strconv.Atoi(mux.Vars(r)["studentID"])
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "The student id must be an integer")
		return
	}

	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Client().Disconnect(context.Background())

	var apps []Application
	cur, err := conn.Collection("applications").Find(context.TODO(), bson.M{"studentID": studentID, "type": "web", "isPublic": true})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Errorf(w, http.StatusNotFound, "student with id %d not found", studentID)
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
		return
	}
	err = cur.All(context.TODO(), &apps)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
		return
	}
	resp.SuccessParse(w, http.StatusOK, fmt.Sprintf("Public applications of %d", studentID), apps)
}

func (h Handler) PublishApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the student from the cookies, get the application from the database
	//check if the student is the owner of the application, if so update the scope to public

	containerId := mux.Vars(r)["containerID"]
	//connect to database
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//get the application from the database
	_, err = conn.Collection("applications").UpdateOne(context.TODO(), bson.M{"containerID": containerId, "studentID": student.ID}, bson.M{"$set": bson.M{"isPublic": true}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Error(w, http.StatusNotFound, "No application found, check if the container id is correct or make sure you own this container")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}
	resp.Success(w, http.StatusOK, "Application published")
}

func (h Handler) RevokeApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the student from the cookies, get the application from the database
	//check if the student is the owner of the application, if so update the scope to public

	containerId := mux.Vars(r)["containerID"]
	//connect to database
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	_, err = conn.Collection("applications").UpdateOne(context.TODO(), bson.M{"containerID": containerId, "studentID": student.ID}, bson.M{"$set": bson.M{"isPublic": false}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Error(w, http.StatusNotFound, "No application found, check if the container id is correct or make sure you own this container")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}
	resp.Success(w, http.StatusOK, "Application is now private")
}
