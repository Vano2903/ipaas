package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

/*
!PAGES ENDPOINTS:
*endpoints for public pages:
/static -> static files
/ -> homepage
/login -> login page
/{studentID} -> public student page
/{studentID}/{appID} -> public app page with info of it

*endpoints for student pages:
/user/ -> user area
/user/application/new -> page to create a new application
/user/database/new -> page to create a new database
! not implemented as pages  /user/application/new -> new application page
! not implemented as pages  /user/database/new -> new database page

!API ENDPOINTS:
*public api endpoints:
/api/oauth -> oauth endpoint (generate the oauth url)
/api/tokens/new -> get a new token pair from a refresh token
/api/{studentID}/all -> get all the public applications of a student
/api/{studentID}/{appID} -> get the info of a public application

*user api endpoints:
/api/user/ -> get the info of the user
/api/user/getApps/{type} -> get all the applications of a user (private or public) with the type of application (database, web, all, updatable)

*container api endpoints:
/api/container/delete/{containerID} -> delete a container
/api/container/publish/{containerID} -> publish a container
/api/container/revoke/{containerID} -> revoke a container

*api endpoints for database:
/api/db/new -> create a new database
! not implemented /api/db/export/{containerID}/{dbName} -> export a database

*api endpoints for applications:
/api/app/new -> create a new application
/api/app/update/{containerID} -> update an application if the repo is changed
*/

func main() {
	mainRouter := mux.NewRouter()
	mainRouter.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("static/"))))

	mainRouter.HandleFunc("/", handler.UserPageHandler)
	mainRouter.HandleFunc("/login", handler.LoginPageHandler)
	mainRouter.HandleFunc("/{studentID}", handler.PublicStudentPageHandler)
	// mainRouter.HandleFunc("/{studentID}/{appID}", handler.PublicAppPageHandler)

	//homepage for the logged user
	mainRouter.HandleFunc("/user/", handler.UserPageHandler).Methods("GET")

	//user's router with access token middleware
	userAreaRouter := mainRouter.PathPrefix("/user").Subrouter()
	//set middleware on user area router
	userAreaRouter.Use(handler.TokensMiddleware)
	//page to create a new application
	userAreaRouter.HandleFunc("/application/new", handler.NewAppPageHandler).Methods("GET")
	//page to create a new database
	userAreaRouter.HandleFunc("/database/new", handler.NewDatabasePageHandler).Methods("GET")

	//!API HANDLERS
	api := mainRouter.PathPrefix("/api").Subrouter()

	//! PUBLIC HANDLERS
	api.HandleFunc("/oauth", handler.OauthHandler).Methods("GET")
	api.HandleFunc("/oauth/check/{randomID}", handler.CheckOauthState).Methods("GET")
	api.HandleFunc("/tokens/new", handler.NewTokenPairFromRefreshTokenHandler).Methods("GET")
	api.HandleFunc("/{studentID}/all", handler.GetAllApplicationsOfStudentPublic).Methods("GET")
	// api.HandleFunc("/{studentID}/{appID}", handler.GetInfoApplication).Methods("GET")

	//! USER HANDLERS
	userApiRouter := api.PathPrefix("/user").Subrouter()
	userApiRouter.Use(handler.TokensMiddleware)
	//get the user data
	//still kinda don't know what to do with this one, will probably return the homepage
	userApiRouter.HandleFunc("/", handler.LoginHandler).Methods("GET")
	//validate a GitHub repo and (if valid) returns the branches
	userApiRouter.HandleFunc("/validate", handler.ValidGithubUrlAndGetBranches).Methods("POST")
	//get all the applications (even the private one) must define the type (database, web, all)
	userApiRouter.HandleFunc("/getApps/{type}", handler.GetAllApplicationsOfStudentPrivate).Methods("GET")
	//update an application
	userApiRouter.HandleFunc("/application/update/{containerID}", handler.UpdateApplicationHandler).Methods("GET")

	//! CONTAINER HANDLERS
	containerApiRouter := api.PathPrefix("/container").Subrouter()
	containerApiRouter.Use(handler.TokensMiddleware)
	//delete a container
	containerApiRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")
	//publish a container
	containerApiRouter.HandleFunc("/publish/{containerID}", handler.PublishApplicationHandler).Methods("GET")
	//revoke a container
	containerApiRouter.HandleFunc("/revoke/{containerID}", handler.RevokeApplicationHandler).Methods("GET")

	//! DBaaS HANDLERS
	//DBaaS router (subrouter of user area router so it has access token middleware)
	dbApiRouter := api.PathPrefix("/db").Subrouter()
	dbApiRouter.Use(handler.TokensMiddleware)
	//let the user create a new database
	dbApiRouter.HandleFunc("/new", handler.NewDBHandler).Methods("POST")
	//export a database
	//dbApiRouter.HandleFunc("/export/{containerID}/{dbName}")

	//! APPLICATIONS HANDLERS
	//application router, it's the main part of the application
	appApiRouter := api.PathPrefix("/app").Subrouter()
	appApiRouter.Use(handler.TokensMiddleware)
	appApiRouter.HandleFunc("/new", handler.NewApplicationHandler).Methods("POST")
	appApiRouter.HandleFunc("/update/{containerID}", handler.UpdateApplicationHandler).Methods("POST")

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "POST"})

	//start event handler
	log.Println("starting event handler")
	go handler.cc.EventHandler()

	log.Println("starting the server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(mainRouter)))
}
