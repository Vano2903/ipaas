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

	r.HandleFunc("/oauth", handler.OauthHandler)

	//user's router with access token middleware
	userAreaRouter := r.PathPrefix("/user").Subrouter()
	//set middleware on user area router
	userAreaRouter.Use(handler.TokensMiddleware)
	userAreaRouter.HandleFunc("/login", handler.LoginHandler)

	//dbaas router (subrouter of user area router so it has access token middleware)
	dbRouter := userAreaRouter.PathPrefix("/db").Subrouter()
	dbRouter.HandleFunc("/new", handler.NewDBHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}

	log.Println("starting the server on port 8080")
	log.Fatal(server.ListenAndServe())
}
