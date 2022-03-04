package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

//struct used to get the user from paleoid
type Payload struct {
	GrantType    string `json:"grant_type"`    //will always be "authorization_code"
	Code         string `json:"code"`          //the code returned by the oauth server
	RedirectUri  string `json:"redirect_uri"`  //the redirect uri (saved in env variable)
	ClientID     string `json:"client_id"`     //the client id (saved in env variable)
	ClientSecret string `json:"client_secret"` //the client secret (saved in env variable)
}

//!extremly important, must modify the use of a session to store the tokens and use cookies

/*
this middleware validates if the users has a valid access token
what it does/check:
1) if the session is found
2) if the refresh token is found (if not it will redirect to the oauth server)
3) if the refresh token is expired
4) if the access token is found/valid, since this runs after the check of the refresh token
the middleware will generate a new pair of tokens if the access token is expired/not found and
save them in the session
5) if it pass all the check it will run the func given as parameter
*/
func AccessTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//get the session
		session, err := store.Get(r, "ipaas-session")
		if err != nil {
			//!should redirect to the oauth page
			returnError(w, http.StatusBadRequest, "unable to get session cookie")
			return
		}

		//create a connection with the db
		db, err := connectToDB()
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer db.Close()

		//get the tokens from the session (interfaces)
		accessTokenInterface := session.Values["accessToken"]
		refreshTokenInterface := session.Values["refreshToken"]

		//check if the refresh token is found
		if refreshTokenInterface == nil {
			//!should redirect to the oauth page
			returnError(w, http.StatusBadRequest, "No refresh token, you should visit the oauth page")
			return
		}

		//check if the refresh token is expired
		if IsTokenExpired(false, refreshTokenInterface.(string), db) {
			//!should redirect to the oauth page
			returnError(w, http.StatusBadRequest, "Refresh token is expired")
			return
		}

		accessToken := accessTokenInterface.(string)
		//check if the access token is found and valid
		if accessTokenInterface == nil || IsTokenExpired(true, accessToken, db) {
			var refreshToken string
			//generate a new token pair
			accessToken, refreshToken, err = GenerateNewTokenPairFromRefreshToken(refreshTokenInterface.(string), db)
			if err != nil {
				returnError(w, http.StatusInternalServerError, err.Error())
				return
			}
			//save them in the session
			session.Values["accessToken"] = accessToken
			session.Values["refreshToken"] = refreshToken
			session.Save(r, w)
		}

		//redirect to the actual handler
		next.ServeHTTP(w, r)
	})
}

//! still under development
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

//oauth handler, will handle the 2 steps of the oauth process
func OauthHandler(w http.ResponseWriter, r *http.Request) {
	//connect to the db
	db, err := connectToDB()
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()
	session, _ := store.Get(r, "ipaas-session")
	//read url parameters
	parameters := r.URL.Query()
	UrlCode, okCode := parameters["code"]
	UrlState, okState := parameters["state"]

	//check if a paleoid access token is stored in the session
	if session.Values["paleoidAccessToken"] != nil {
		paleoIDAccessToken := session.Values["paleoidAccessToken"].(string)
		//register the user (if not alreayd registered) from the access token and generate a token pain
		resp, isClientSide, err := registerOrGenerateTokenFromPaleoIDAccessToken(paleoIDAccessToken, db)
		if err != nil {
			if isClientSide {
				returnError(w, http.StatusBadRequest, err.Error())
			} else {
				returnError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		//save the tokens in the session
		session.Values["accessToken"] = resp["accessToken"]
		session.Values["refreshToken"] = resp["refreshToken"]
		session.Save(r, w)
		returnSuccessMap(w, http.StatusOK, "Token generated", resp)
		return
	}

	//check if it's the second phase of the oauth
	if okCode && okState {
		//check if the state is valid (rsa envryption)
		valid, err := CheckState(UrlState[0])
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !valid {
			returnError(w, http.StatusBadRequest, "Invalid state")
			return
		}

		//get the paleoid access token
		paleoidAccessToken, err := GetPaleoIDAccessToken(UrlCode[0])
		if err != nil {
			returnError(w, http.StatusInternalServerError, err.Error())
			return
		}

		//use this paleoid to generate a token pair and save the user on the db in case he is not already registered
		resp, isClientSide, err := registerOrGenerateTokenFromPaleoIDAccessToken(paleoidAccessToken, db)
		if err != nil {
			if isClientSide {
				returnError(w, http.StatusBadRequest, err.Error())
			} else {
				returnError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		//save the tokens in the session
		session.Values["accessToken"] = resp["accessToken"]
		session.Values["refreshToken"] = resp["refreshToken"]
		session.Save(r, w)
		returnSuccessMap(w, http.StatusOK, "Token generated", resp)
		return
	}

	//check if a server generated state is stored in the session
	if session.Values["state"] != nil {
		oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), session.Values["state"], os.Getenv("REDIRECT_URI"))
		// http.Redirect(w, r, oauthUrl, http.StatusFound)
		returnSuccess(w, http.StatusOK, oauthUrl)
		return
	}
	//generate a new base64url encoded signed with rsa encrypted state (random string) and stored on the db (plain)
	state, err := CreateState()
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	//set the state on the session
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

//get the user's informations from the ipaas access token
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectToDB()
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	session, _ := store.Get(r, "ipaas-session")
	accessToken := session.Values["accessToken"].(string)
	student, err := GetUserFromAccessToken(accessToken, db)
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	studentByte, err := json.MarshalIndent(student, "", "  ")
	if err != nil {
		returnError(w, http.StatusInternalServerError, err.Error())
		return
	}
	returnSuccessJson(w, http.StatusOK, "User", studentByte)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/oauth", OauthHandler)

	//user's router with access token middleware
	userAreaRouter := r.PathPrefix("/user").Subrouter()
	//set middleware on user area router
	userAreaRouter.Use(AccessTokenMiddleware)
	userAreaRouter.HandleFunc("/login", LoginHandler)

	//dbaas router (subrouter of user area router so it has access token middleware)
	dbRouter := userAreaRouter.PathPrefix("/db").Subrouter()
	dbRouter.HandleFunc("/new", NewDBHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}

	log.Println("starting the server on port 8080")
	log.Fatal(server.ListenAndServe())
}
