package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	mainRouter := mux.NewRouter()
	mainRouter.Handle("/static", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	mainRouter.HandleFunc("/", handler.homePageHandler)
	mainRouter.HandleFunc("/login", handler.loginPageHandler)
	mainRouter.HandleFunc("/{studentID}", handler.publicStudentPageHandler)
	mainRouter.HandleFunc("/{studentID}/{appID}", handler.publicAppPageHandler)

	//user's router with access token middleware
	userAreaRouter := mainRouter.PathPrefix("/user").Subrouter()
	//set middleware on user area router
	userAreaRouter.Use(handler.TokensMiddleware)
	//homepage
	userAreaRouter.HandleFunc("/", handler.userPageHandler).Methods("GET")
	//page to create a new application
	userAreaRouter.HandleFunc("/application/new", handler.newAppPageHandler).Methods("GET")
	//page to create a new database
	userAreaRouter.HandleFunc("/database/new", handler.newDatabasePageHandler).Methods("GET")

	//!API HANDLERS
	api := mainRouter.PathPrefix("/api").Subrouter()

	//! PUBLIC HANDLERS
	api.HandleFunc("/oauth", handler.OauthHandler).Methods("GET")
	api.HandleFunc("/tokens/new", handler.NewTokenPairFromRefreshTokenHandler).Methods("GET")
	api.HandleFunc("/{studentID}/all", handler.GetAllApplicationsOfStudentPublic).Methods("GET")
	api.HandleFunc("/{studentID}/{appID}", handler.GetInfoApplication).Methods("GET")

	//! USER HANDLERS
	//get the user data
	//still kinda don't know what to do with this one, will probably return the homepage
	userAreaRouter.HandleFunc("/", handler.LoginHandler).Methods("GET")
	//get all the applications (even the private one) must define the type (database, web, all)
	userAreaRouter.HandleFunc("/getApps/{type}", handler.GetAllApplicationsOfStudentPrivate).Methods("GET")

	//! DBaaS HANDLERS
	//DBaaS router (subrouter of user area router so it has access token middleware)
	dbRouter := api.PathPrefix("/db").Subrouter()
	dbRouter.Use(handler.TokensMiddleware)
	//let the user create a new database
	dbRouter.HandleFunc("/new", handler.NewDBHandler).Methods("POST")
	//delete a database
	dbRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")
	// dbRouter.HandleFunc("/export/{containerID}/{dbName}")

	//! APPLICATIONS HANDLERS
	//application router, it's the main part of the application
	appRouter := api.PathPrefix("/app").Subrouter()
	api.Use(handler.TokensMiddleware)
	appRouter.HandleFunc("/new", handler.NewApplicationHandler).Methods("POST")
	appRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")
	appRouter.HandleFunc("/update/{containerID}", handler.UpdateApplicationHandler).Methods("POST")

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mainRouter,
	}

	log.Println("starting the server on port 8080")
	log.Fatal(server.ListenAndServe())
}
