package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	resp "github.com/vano2903/ipaas/responser"
)

type Post struct {
	GithubRepoUrl string `json:"github-repo"`
	GithubBranch  string `json:"github-branch"`
	Language      string `json:"language"`
	Port          string `json:"port"`
}

type Handler struct {
	cc   *ContainerController
	sess *sessions.CookieStore
	util *Util
}

//!===========================MIDDLEWARES

//check if the user has a valid access Token
func (h Handler) TokensMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("tokens middleware")

		// get the access token from cookies
		var accessToken string
		for _, cookie := range r.Cookies() {
			switch cookie.Name {
			case "accessToken":
				accessToken = cookie.Value
			}
		}

		//check if it's not empty
		//498 => token invalid/expired
		if accessToken == "" {
			resp.Error(w, 498, "No access token")
			return
		}

		//create a connection with the db
		db, err := connectToDB()
		if err != nil {
			resp.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer db.Close()

		//check if it's expired
		if IsTokenExpired(true, accessToken, db) {
			resp.Error(w, 498, "Access token is expired")
			return
		}

		//redirect to the actual handler
		next.ServeHTTP(w, r)
	})
}

//!===========================DATABASE RELATED HANDLERS

//! still under development
//new database handler, let the user create a new database given the default db name (can be null)
//credentials will be autogenerated
//the body should contain:
//1) db name
//2) db type (mysql, mariadb, mongodb)
//3) (not implemented) db version (can be null which will mean the latest version)
func (h Handler) NewDBHandler(w http.ResponseWriter, r *http.Request) {
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
	var dbPost dbPost
	err = json.NewDecoder(r.Body).Decode(&dbPost)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error decoding the json: %v", err.Error())
		return
	}

	//generate env variables for the container
	password := generateRandomString(16)
	var env []string
	switch dbPost.DbType {
	case "mysql", "mariadb":
		//root password
		env = []string{
			"MYSQL_ROOT_PASSWORD=" + password,
		}
		//set a database if it's not empty
		if dbPost.DbName != "" {
			env = append(env, "MYSQL_DATABASE="+dbPost.DbName)
		}
	case "mongodb":
		//root password
		env = []string{
			"MONGO_INITDB_ROOT_USERNAME=" + "root",
			"MONGO_INITDB_ROOT_PASSWORD=" + password,
		}
		//set a database if it's not empty
		//!apparently this part doesn't work, the user can create the db on it's own though
		if dbPost.DbName != "" {
			env = append(env, "MONGO_INITDB_DATABASE="+dbPost.DbName)
		}
	default:
		resp.Error(w, http.StatusBadRequest, "Invalid db type, must be mysql, mariadb or mongodb")
		return
	}

	//create the database container
	id, err := h.cc.CreateNewDB(h.cc.dbContainersConfigs[dbPost.DbType], env)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error creating a new database: %v", err.Error())
		return
	}

	//get the external port
	port, err := h.cc.GetContainerExternalPort(id, h.cc.dbContainersConfigs[dbPost.DbType].port)
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error getting the external port: %v", err.Error())
		return
	}

	//add a new database application created by the student (student id)
	insertApplicationQuery := `
	INSERT INTO applications (containerID, status, studentID, type, name, description) VALUES (?, ?, ?, ?, ?, ?)
	`

	//exec the query, the status will be up and type database, the name will follow the nomenclature of <studentID>:<dbType>/<dbName>
	_, err = conn.Exec(insertApplicationQuery, id, "up", student.ID, "database", fmt.Sprintf("%d:%s/%s", student.ID, dbPost.DbType, dbPost.DbName), "")
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "unable to link the application to the user: %v", err.Error())
		return
	}

	json := make(map[string]interface{})

	json = map[string]interface{}{
		"important": "the password is for root user of the server",
		"user":      "root",
		"port":      port,
		"pass":      password,
	}
	if dbPost.DbType == "mongodb" {
		json["uri"] = fmt.Sprintf("mongodb://root:%s@%s:%s", password, "127.0.0.1", port)
	}
	resp.SuccessParse(w, http.StatusOK, "New DB created", json)
}

//!===========================APPLICATIONS RELATED HANDLERS

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
	var appPost Post
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
	// repo, name, err := h.util.DownloadGithubRepo(student.ID, appPost.GithubBranch, appPost.GithubRepoUrl)
	// if err != nil {
	// 	resp.Errorf(w, http.StatusInternalServerError, "error downloading the repo, try again in one minute: %v", err.Error())
	// 	return
	// }

	// fmt.Println("repo:", repo)
	// fmt.Println("name:", name)

	repo := "tmp/18008-testing"
	name := "testing"

	//port to expose for the app
	port, err := strconv.Atoi(appPost.Port)
	if err != nil {
		resp.Errorf(w, http.StatusBadRequest, "error converting the port to an int: %v", err.Error())
		return
	}

	//create the image from the repo downloaded
	imageName, imageID, err := h.cc.CreateImage(student.ID, port, name, repo, appPost.Language, nil)
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

	//remove the repo after creating the application
	// err = os.RemoveAll(repo)
	// if err != nil {
	// 	resp.Errorf(w, http.StatusInternalServerError, "error removing the repo: %v", err)
	// 	return
	// }

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

	toSend := map[string]interface{}{
		"container id":  id,
		"external port": exernalPort,
		"status":        status,
	}

	// logs, _ := h.cc.GetContainerLogs(id)
	// fmt.Println("logs: ", logs)

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
	var app Application
	err = conn.QueryRow(`SELECT * FROM applications WHERE studentID = ? AND containerID = ?`, student.ID, containerID).Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &app.CreatedAt, &app.IsPublic)
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
	resp.Success(w, http.StatusOK, "container deleted successfully")
}

//it will return a json with all the applications owned by the student (even the privates one)
//this endpoint will only be accessible if logged in
func (h Handler) GetAllApplicationsOfStudentPrivate(w http.ResponseWriter, r *http.Request) {
	//get the {type} of the application from the url
	typeOfApp := mux.Vars(r)["type"]
	if typeOfApp != "database" && typeOfApp != "web" && typeOfApp != "all" {
		resp.Error(w, http.StatusBadRequest, "Invalid type of application")
		return
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
	for rows.Next() {
		var app Application
		err = rows.Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &app.CreatedAt, &app.IsPublic)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		applications = append(applications, app)
	}

	resp.SuccessParse(w, http.StatusOK, "Applications", applications)
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
		err = rows.Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description, &app.CreatedAt, &app.IsPublic)
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the applications: %v", err.Error())
			return
		}
		applications = append(applications, app)
	}
	resp.SuccessParse(w, http.StatusOK, "Public applications of "+studentID, applications)
}

//!===========================GENERICS HANDLERS

//oauth handler, will handle the 2 steps of the oauth process
//all the procedure is in https://paleoid.stoplight.io/docs/api/YXBpOjQxNDY4NTk-paleo-id-o-auth2-api
func (h Handler) OauthHandler(w http.ResponseWriter, r *http.Request) {
	//connect to the db
	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()
	session, _ := h.sess.Get(r, "ipaas-session")
	//read url parameters (code and state)
	parameters := r.URL.Query()
	UrlCode, okCode := parameters["code"]
	UrlState, okState := parameters["state"]

	//check if a paleoid access token is stored in the session
	if session.Values["paleoidAccessToken"] != nil {
		paleoIDAccessToken := session.Values["paleoidAccessToken"].(string)
		//register the user (if not alreayd registered) from the access token and generate a token pain
		response, isClientSide, err := registerOrGenerateTokenFromPaleoIDAccessToken(paleoIDAccessToken, db)
		if err != nil {
			if isClientSide {
				resp.Error(w, http.StatusBadRequest, err.Error())
			} else {
				resp.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		//save the tokens in the session
		//set the tokens as cookie
		//!should set domain and path
		http.SetCookie(w, &http.Cookie{
			Name:    "accessToken",
			Value:   response["accessToken"].(string),
			Expires: time.Now().Add(time.Hour),
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "refreshToken",
			Value:   response["refreshToken"].(string),
			Expires: time.Now().Add(time.Hour * 24 * 7),
		})
		resp.SuccessParse(w, http.StatusOK, "Token generated", response)
		return
	}

	//check if it's the second phase of the oauth
	if okCode && okState {
		//check if the state is valid (rsa envryption)
		valid, err := CheckState(UrlState[0])
		if err != nil {
			resp.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !valid {
			resp.Error(w, http.StatusBadRequest, "Invalid state")
			return
		}

		//get the paleoid access token
		paleoidAccessToken, err := GetPaleoIDAccessToken(UrlCode[0])
		if err != nil {
			resp.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		//use this paleoid to generate a token pair and save the user on the db in case he is not already registered
		response, isClientSide, err := registerOrGenerateTokenFromPaleoIDAccessToken(paleoidAccessToken, db)
		if err != nil {
			if isClientSide {
				resp.Error(w, http.StatusBadRequest, err.Error())
			} else {
				resp.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		//save the tokens in the session

		//!should set domain and path
		http.SetCookie(w, &http.Cookie{
			Name:    "accessToken",
			Value:   response["accessToken"].(string),
			Expires: time.Now().Add(time.Hour),
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "refreshToken",
			Value:   response["refreshToken"].(string),
			Expires: time.Now().Add(time.Hour * 24 * 7),
		})
		resp.SuccessParse(w, http.StatusOK, "Token generated", response)
		return
	}

	//check if a server generated state is stored in the session
	if session.Values["state"] != nil {
		oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), session.Values["state"], os.Getenv("REDIRECT_URI"))
		// http.Redirect(w, r, oauthUrl, http.StatusFound)
		resp.Success(w, http.StatusOK, oauthUrl)
		return
	}
	//generate a new base64url encoded signed with rsa encrypted state (random string) and stored on the db (plain)
	state, err := CreateState()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	//set the state on the session
	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		log.Println(err)
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), state, os.Getenv("REDIRECT_URI"))
	resp.Success(w, http.StatusOK, oauthUrl)
}

//get the user's informations from the ipaas access token
func (h Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	log.Println("getting access token")
	//get the access token from the cookie
	cookie, err := r.Cookie("accessToken")
	if err != nil {
		if err == http.ErrNoCookie {
			resp.Error(w, http.StatusBadRequest, "No access token")
			return
		}
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	accessToken := cookie.Value
	log.Println("access token found:", accessToken)

	//get the student generic infos from the access token
	student, err := GetUserFromAccessToken(accessToken, db)
	fmt.Println("studente:", student)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp.SuccessParse(w, http.StatusOK, "User", student)
}

//generate a new token pair from the refresh token saved in the cookies
func (h Handler) NewTokenPairFromRefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	//get the refresh token from the cookie
	cookie, err := r.Cookie("refreshToken")
	if err != nil {
		if err == http.ErrNoCookie {
			resp.Error(w, http.StatusBadRequest, "No refresh token")
			return
		}
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	refreshToken := cookie.Value
	log.Println("refresh token found:", refreshToken)

	//check if there is a refresh token
	if refreshToken == "" {
		resp.Error(w, 498, "No refresh token")
		return
	}

	//connection to db
	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	//check if the refresh token is expired
	if IsTokenExpired(false, refreshToken, db) {
		//!should redirect to the oauth page
		resp.Error(w, 498, "Refresh token is expired")
		return
	}

	//generate a new token pair
	accessToken, newRefreshToken, err := GenerateNewTokenPairFromRefreshToken(refreshToken, db)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	//delete the old tokens from the cookies
	http.SetCookie(w, &http.Cookie{
		Name:    "accessToken",
		Path:    "/",
		Value:   "",
		Expires: time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "refreshToken",
		Path:    "/",
		Value:   "",
		Expires: time.Unix(0, 0),
	})

	//set the new tokens
	//!should set domain and path
	http.SetCookie(w, &http.Cookie{
		Name:    "accessToken",
		Path:    "/",
		Value:   accessToken,
		Expires: time.Now().Add(time.Hour),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "refreshToken",
		Path:    "/",
		Value:   newRefreshToken,
		Expires: time.Now().Add(time.Hour * 24 * 7),
	})

	//we also respond with the new tokens so the client doesn't have to depend from the cookies
	response := map[string]interface{}{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
	}
	resp.SuccessParse(w, http.StatusOK, "New token pair generated", response)
}

//constructor
func NewHandler() (*Handler, error) {
	var h Handler
	var err error
	h.sess = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	h.cc, err = NewContainerController()
	if err != nil {
		return nil, err
	}
	h.util, err = NewUtil()
	if err != nil {
		return nil, err
	}
	return &h, nil
}

//!BACKLOG, UNDER DEVELOPMENT

//! under development, not implementing it now
//! if you want to export the database use an external tool
// /user/
// func (h Handler) ExportDBHandler(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	containerID := vars["containerID"]

// 	//read the post body
// 	var dbPost dbPost
// 	err := json.NewDecoder(r.Body).Decode(&dbPost)
// 	if err != nil {
// 		resp.Errorf(w, http.StatusBadRequest, "error decoding the json: %v", err.Error())
// 		return
// 	}

// 	//connect to the db
// 	conn, err := connectToDB()
// 	if err != nil {
// 		resp.Errorf(w, http.StatusInternalServerError, "error connecting to the database: %v", err.Error())
// 		return
// 	}
// 	defer conn.Close()
// 	//get the student from the cookies
// 	student, err := h.util.GetUserFromCookie(r, conn)
// 	if err != nil {
// 		resp.Errorf(w, http.StatusInternalServerError, "error getting the user from cookies: %v", err.Error())
// 		return
// 	}

// 	//check if the user owns the db container
// 	var app Application
// 	err = conn.QueryRow(`SELECT * FROM applications WHERE type='database' AND studentID = ? AND containerID = ?`, student.ID, containerID).Scan(&app.ID, &app.ContainerID, &app.Status, &app.StudentID, &app.Type, &app.Name, &app.Description)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			resp.Error(w, http.StatusBadRequest, "application not found, check if the container id is correct and make sure you own this database")
// 		}
// 	}

// 	dbPost.DbType = strings.Split(strings.Split(app.Name, ":")[1], "/")[0]

// }
