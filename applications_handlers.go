package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
	resp "github.com/vano2903/ipaas/responser"
)

//TODO: should delete the container/image in case of an error because it would stop the user to create again the application
//new application handler let the user host a new application given:
//1) github repository
//2) programming lang
//3) port of the program
//! for now the only supported applications are web based one
func (h Handler) NewApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Close()

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

	//check that the appPost.GithubRepo is an actual url
	if !h.util.ValidGithubUrl(appPost.GithubRepoUrl) {
		resp.Error(w, http.StatusBadRequest, "Invalid github repo url")
		return
	}

	//download the repo
	repo, name, hash, err := h.util.DownloadGithubRepo(student.ID, appPost.GithubBranch, appPost.GithubRepoUrl)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error downloading the repo, try again in one minute: %v", err.Error())
		return
	}
	fmt.Println("repo: ", repo)
	fmt.Println("name: ", name)
	fmt.Println("hash: ", hash)

	//port to expose for the app
	port, err := strconv.Atoi(appPost.Port)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error converting the port to an int: %v", err.Error())
		return
	}

	//create the image from the repo downloaded
	imageName, imageID, err := h.cc.CreateImage(student.ID, port, name, repo, appPost.Language, appPost.Envs)
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
	err = os.RemoveAll(repo)
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

	//add a new database application created by the student (student id)
	insertApplicationQuery := `
	INSERT INTO applications 
	(containerID, status, studentID, type, name, description, githubRepo, lastCommit, port, language, externalPort) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	//exec the query, the status will be up and type database, the name will follow the nomenclature of <studentID>:<dbType>/<dbName>
	app, err := conn.Exec(insertApplicationQuery, id, status, student.ID, "web", imageName, appPost.Description, appPost.GithubRepoUrl, hash, appPost.Port, appPost.Language, exernalPort)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "unable to link the application to the user: %v", err.Error())
		return
	}

	appId, err := app.LastInsertId()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "unable to link the application to the user: %v", err.Error())
		return
	}

	insertEnvsQuery := `INSERT INTO envs (applicationID, key, value) VALUES (?, ?, ?)`
	for key, value := range appPost.Envs {
		_, err = conn.Exec(insertEnvsQuery, appId, key, value)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "unable to insert the env: %v", err.Error())
			return
		}
	}

	toSend := map[string]interface{}{
		"container id":  id,
		"external_port": os.Getenv("IP") + ":" + exernalPort,
		"status":        status,
	}

	resp.SuccessParse(w, http.StatusOK, "application created", toSend)
}

//delete a container given the container id, it will check if the user owns this application
func (h Handler) DeleteApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the container from /{containerID}
	containerID := mux.Vars(r)["containerID"]

	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Close()

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//get the application from the database and check if it's owned by the student
	// var app Application
	// var tmp, githubRepo, LastCommitHash, port, Lang sql.NullString
	var id int
	err = conn.QueryRow(`SELECT id FROM applications WHERE studentID = ? AND containerID = ?`, student.ID, containerID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Errorf(w, http.StatusBadRequest, "application not found, the application might have the wrong container id or you dont own it")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}

	//delete the container
	err = h.cc.DeleteContainer(containerID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the container: %v", err.Error())
		return
	}

	//delete the application from the database
	_, err = conn.Exec(`DELETE FROM applications WHERE containerID = ?`, containerID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the application: %v", err.Error())
		return
	}

	_, err = conn.Exec(`DELETE FROM envs WHERE applicationID = ?`, id)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting the envs: %v", err.Error())
		return
	}
	resp.Success(w, http.StatusOK, "container deleted successfully")
}

//!hold the port for the new container
//!dont use a different one
//*1) delete the old container
//*2) download the repo from github
//*3) build the image
//*4) create the new container
//*5) cleanups
//! should delete old container lastly but for i gotta change way i create images
func (h Handler) UpdateApplicationHandler(w http.ResponseWriter, r *http.Request) {
	//get the container from /{containerID}
	containerID := mux.Vars(r)["containerID"]

	//connect to the db
	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Close()

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//get the application from the database and check if it's owned by the student
	var app Application
	var githubRepo, LastCommitHash, branch, port, Lang, externalPort sql.NullString
	err = conn.QueryRow(`SELECT * FROM applications WHERE studentID = ? AND containerID = ? AND type = "web"`, student.ID, containerID).Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &githubRepo, &LastCommitHash, &branch, &port, &externalPort, &Lang, &app.CreatedAt, &app.IsPublic)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Errorf(w, http.StatusBadRequest, "application not found, the application might have the wrong container id or you dont own it")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}

	app.GithubRepo = githubRepo.String
	app.LastCommitHash = LastCommitHash.String
	app.GithubBranch = branch.String
	app.Port = port.String
	app.Lang = Lang.String
	app.ExternalPort = externalPort.String

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

	//get the environment variables
	envs := make(map[string]string)
	rows, err := conn.Query(`SELECT * FROM envs WHERE applicationID = ?`, app.ID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the envs: %v", err.Error())
		return
	}
	for rows.Next() {
		var tmp int
		var key, value string
		err = rows.Scan(&tmp, &key, &value)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the envs: %v", err.Error())
			return
		}
		envs[key] = value
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
	imageName, _, err := h.cc.CreateImage(student.ID, Intport, name, repo, app.Lang, envs)
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

	//add a new database application created by the student (student id)
	updateApplicationQuery := `
	UPDATE applications 
	SET status = ?, containerID = ?, lastCommit = ?, externalPort = ? 
	WHERE id = ?`

	//exec the query, the status will be up and type database, the name will follow the nomenclature of <studentID>:<dbType>/<dbName>
	_, err = conn.Exec(updateApplicationQuery, status, newContainerID, hash, exernalPort, app.ID)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "unable to link the application to the user: %v", err.Error())
		return
	}

	toSend := map[string]interface{}{
		"container id":  newContainerID,
		"external port": os.Getenv("IP") + ":" + exernalPort,
		"status":        status,
	}

	resp.SuccessParse(w, http.StatusOK, "application updated", toSend)
}

//it will return a json with all the applications owned by the student (even the privates one)
//this endpoint will only be accessible if logged in
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
	defer conn.Close()

	//get the student from the cookies
	student, err := h.util.GetUserFromCookie(r, conn)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
		return
	}

	//get the applications from the database
	var rows *sql.Rows
	if typeOfApp == "all" {
		rows, err = conn.Query("SELECT * FROM applications WHERE studentID = ?", student.ID)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
	} else {
		rows, err = conn.Query("SELECT * FROM applications WHERE studentID = ? AND type = ?", student.ID, typeOfApp)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
	}

	//parse the rows into a []Application and return it
	var applications []Application
	var errors []string
	for rows.Next() {
		var app Application
		//tmp is branch
		var tmp, githubRepo, LastCommitHash, port, Lang, externalPort sql.NullString

		err = rows.Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &githubRepo, &LastCommitHash, &tmp, &port, &externalPort, &Lang, &app.CreatedAt, &app.IsPublic)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		app.GithubRepo = githubRepo.String
		app.LastCommitHash = LastCommitHash.String
		app.Port = port.String
		app.Lang = Lang.String
		app.ExternalPort = os.Getenv("IP") + ":" + externalPort.String
		app.IsUpdatable, err = h.util.HasLastCommitChanged(app.LastCommitHash, app.GithubRepo, app.GithubBranch)
		if err != nil {
			app.IsUpdatable = false
			errors = append(errors, err.Error())
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

//get all the public applications of a student given the student id
//you dont need to be logged in to access this endpoint
func (h Handler) GetAllApplicationsOfStudentPublic(w http.ResponseWriter, r *http.Request) {
	//get the {studentID} from the url
	studentID := mux.Vars(r)["studentID"]

	conn, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
		return
	}
	defer conn.Close()

	//get all the public web applications of the student
	rows, err := conn.Query("SELECT * FROM applications WHERE type = 'web' AND isPublic = 1 AND studentID = ?", studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Error(w, http.StatusNotFound, "No applications found, check if the student id is correct")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
		return
	}

	//parse the rows into a []Application and return it
	var applications []Application
	for rows.Next() {
		var app Application
		var tmp, githubRepo, LastCommitHash, port, Lang, externalPort sql.NullString

		err = rows.Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &githubRepo, &LastCommitHash, &tmp, &port, &externalPort, &Lang, &app.CreatedAt, &app.IsPublic)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		app.GithubRepo = githubRepo.String
		app.LastCommitHash = LastCommitHash.String
		app.Port = port.String
		app.Lang = Lang.String
		app.ExternalPort = os.Getenv("IP") + ":" + externalPort.String
		applications = append(applications, app)
	}
	resp.SuccessParse(w, http.StatusOK, "Public applications of "+studentID, applications)
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

	var id int
	err = conn.QueryRow("SELECT id FROM applications WHERE containerID = ? AND studentID = ?", containerId, student.ID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Error(w, http.StatusNotFound, "No application found, check if the container id is correct or make sure you own this container")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}

	//update the scope to public
	_, err = conn.Exec("UPDATE applications SET isPublic = 1 WHERE containerID = ?", containerId)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error updating the application: %v", err.Error())
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

	//get the application from the database
	var id int
	err = conn.QueryRow("SELECT id FROM applications WHERE containerID = ? AND studentID = ?", containerId, student.ID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Error(w, http.StatusNotFound, "No application found, check if the container id is correct or make sure you own this container")
			return
		}
		resp.Errorf(w, http.StatusInternalServerError, "error getting the application: %v", err.Error())
		return
	}

	//update the scope to public
	_, err = conn.Exec("UPDATE applications SET isPublic = 0 WHERE containerID = ?", containerId)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error updating the application: %v", err.Error())
		return
	}

	resp.Success(w, http.StatusOK, "Application published")
}
