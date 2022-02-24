package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func NewDBHandler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json")
	password := generatePassword()
	env := []string{
		"MYSQL_ROOT_PASSWORD=" + password,
		"MYSQL_DATABASE=test",
	}
	id, err := c.CreateNewDB(MYSQL_IMAGE, MYSQL_PORT, env)
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	json := map[string]interface{}{
		"id":   id,
		"pass": password,
	}
	returnSuccessJson(w, http.StatusOK, "New DB created", json)
}

func OauthHandler(w http.ResponseWriter, r *http.Request) {
	//read url parameters
	parameters := r.URL.Query()
	UrlCode, okCode := parameters["code"]
	UrlState, okState := parameters["state"]

	if okCode && okState {
		returnSuccess(w, http.StatusOK, fmt.Sprintf("bello :D, code: %s, state: %s", UrlCode, UrlState))
		return
	}

	session, _ := store.Get(r, "ipaas-session")

	fmt.Println(session.Values)

	if session.Values["state"] != nil {
		// returnError(w, http.StatusInternalServerError, "state already exists, either delete it or use it")
		oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), session.Values["state"], os.Getenv("REDIRECT_URI"))
		http.Redirect(w, r, oauthUrl, http.StatusFound)
		return
	}
	state, err := CreateState()
	fmt.Println("state: ", state)
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		log.Println(err)
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	returnSuccess(w, http.StatusOK, state)
	// oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), state, os.Getenv("REDIRECT_URI"))
	// http.Redirect(w, r, oauthUrl, http.StatusFound)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/oauth", OauthHandler)

	dbRouter := r.PathPrefix("/db").Subrouter()
	dbRouter.HandleFunc("/new", NewDBHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r, // Pass our instance of gorilla/mux in.
	}

	log.Fatal(server.ListenAndServe())
}
