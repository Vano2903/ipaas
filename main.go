package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// struct used to get the user from paleoid
type Payload struct {
	GrantType    string `json:"grant_type"`    //will always be "authorization_code"
	Code         string `json:"code"`          //the code returned by the oauth server
	RedirectUri  string `json:"redirect_uri"`  //the redirect uri (saved in env variable)
	ClientID     string `json:"client_id"`     //the client id (saved in env variable)
	ClientSecret string `json:"client_secret"` //the client secret (saved in env variable)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/oauth", handler.OauthHandler).Methods("GET")
	r.HandleFunc("/tokens/new", handler.NewTokenPairFromRefreshTokenHandler).Methods("GET")
	r.HandleFunc("/{studentID}/all", handler.GetAllApplicationsOfStudentPublic).Methods("GET")

	//user's router with access token middleware
	userAreaRouter := r.PathPrefix("/user").Subrouter()
	//set middleware on user area router
	userAreaRouter.Use(handler.TokensMiddleware)
	//get the user data
	//still kinda don't know what to do with this one, will probably return the homepage
	userAreaRouter.HandleFunc("/", handler.LoginHandler).Methods("GET")
	//get all the applications (even the private one) must define the type (database, web, all)
	userAreaRouter.HandleFunc("/getApps/{type}", handler.GetAllApplicationsOfStudentPrivate).Methods("GET")

	//DBaaS router (subrouter of user area router so it has access token middleware)
	dbRouter := userAreaRouter.PathPrefix("/db").Subrouter()
	//let the user create a new database
	dbRouter.HandleFunc("/new", handler.NewDBHandler).Methods("POST")
	//delete a database
	dbRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")
	// dbRouter.HandleFunc("/export/{containerID}/{dbName}")

	//application router, it's the main part of the application
	appRouter := userAreaRouter.PathPrefix("/app").Subrouter()
	// appRouter.HandleFunc("/new", handler.NewAppHandler)
	appRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}

	log.Println("starting the server on port 8080")
	log.Fatal(server.ListenAndServe())
}
