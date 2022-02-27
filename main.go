package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type Payload struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectUri  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewDBHandler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json")
	// password := generatePassword()
	// env := []string{
	// 	"MYSQL_ROOT_PASSWORD=" + password,
	// 	"MYSQL_DATABASE=test",
	// }
	// id, err := c.CreateNewDB(MYSQL_IMAGE, MYSQL_PORT, env)
	// if err != nil {
	// 	returnError(w, http.StatusInternalServerError, err.Error())
	// 	return
	// }
	// json := map[string]interface{}{
	// 	"id":   id,
	// 	"pass": password,
	// }
	// returnSuccessJson(w, http.StatusOK, "New DB created", json)
}

func OauthHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "ipaas-session")
	//read url parameters
	parameters := r.URL.Query()
	UrlCode, okCode := parameters["code"]
	UrlState, okState := parameters["state"]

	fmt.Println("sessioni:", session.Values)

	if okCode && okState {
		//check if the state is valid
		valid, err := CheckState(UrlState[0])
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !valid {
			returnError(w, http.StatusBadRequest, "Invalid state")
			return
		}

		//delete the state from the session
		session.Values["state"] = nil
		// session.Save(r, w)

		//get the access token
		accessToken, err := GetAccessToken(UrlCode[0])
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}

		//get the student
		student, err := GetStudent(accessToken)
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}
		session.Values["accessToken"] = accessToken
		// session.Values["student"] = student
		session.Save(r, w)
		json, _ := json.Marshal(student)
		returnSuccess(w, http.StatusOK, string(json))
		// returnSuccess(w, http.StatusOK, fmt.Sprintf("bello :D, code: %s, state: %s", UrlCode, UrlState))
		return
	}

	if session.Values["state"] != nil {
		oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), session.Values["state"], os.Getenv("REDIRECT_URI"))
		// http.Redirect(w, r, oauthUrl, http.StatusFound)
		returnSuccess(w, http.StatusOK, oauthUrl)
		return
	}
	state, err := CreateState()
	fmt.Println("creating state: ", state)
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

	oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), state, os.Getenv("REDIRECT_URI"))
	returnSuccess(w, http.StatusOK, oauthUrl)
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

	log.Println("starting the server on port 8080")
	log.Fatal(server.ListenAndServe())
}
