package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	resp "github.com/vano2903/ipaas/responser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	cc   *ContainerController
	sess *sessions.CookieStore
	util *Util
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
	defer db.Client().Disconnect(context.TODO())
	session, _ := h.sess.Get(r, "ipaas-session")
	//read url parameters (code and state)
	parameters := r.URL.Query()
	UrlCode, okCode := parameters["code"]
	UrlState, okState := parameters["state"]

	//check if it's the second phase of the oauth
	if okCode && okState {
		//check if the state is valid (rsa envryption)
		valid, redirectUri, err := CheckState(UrlState[0])
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

		ipaasAccessToken := response["ipaas-access-token"].(string)
		ipaasRefreshToken := response["ipaas-refresh-token"].(string)

		randomID, isRandomIDSet := session.Values["randomID"]
		if isRandomIDSet {
			err = updatePollingID(randomID.(string), paleoidAccessToken, ipaasRefreshToken)
			if err != nil {
				resp.Errorf(w, http.StatusInternalServerError, "error update value of pollingID")
				return
			}
		}

		//!should set domain and path
		http.SetCookie(w, &http.Cookie{
			Name:    "ipaas-access-token",
			Path:    "/",
			Value:   ipaasAccessToken,
			Expires: time.Now().Add(time.Hour),
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "ipaas-refresh-token",
			Path:    "/",
			Value:   ipaasRefreshToken,
			Expires: time.Now().Add(time.Hour * 24 * 7),
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "ipaas-session",
			Value:   "",
			Expires: time.Unix(0, 0),
		})

		//if redirect uri is set send a post request with the tokens to that uri
		//if it's empty the token will be shown has a response of the server
		if redirectUri == "" {
			//convert response to post body
			r := struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				UserID       int    `json:"user_id"`
			}{
				ipaasAccessToken,
				ipaasRefreshToken,
				response["userID"].(int),
			}

			//convert r to io.Reader
			body, err := json.Marshal(r)
			if err != nil {
				resp.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			//do post request to the redirect uri sending the body
			bodyBuffer := bytes.NewBuffer(body)
			_, err = http.Post(redirectUri, "application/json", bodyBuffer)
			if err != nil {
				resp.Errorf(w, http.StatusInternalServerError, "error sending port to %s: %v", redirectUri, err.Error())
				return
			}
			resp.Successf(w, http.StatusOK, "Token generated successfully, a post request has been sent to %s", redirectUri)
			return
		}

		resp.SuccessParse(w, http.StatusOK, "Token generated successfully", response)
		return
	}

	//check if a server generated state is stored in the session
	if session.Values["state"] != nil {
		oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), session.Values["state"], os.Getenv("REDIRECT_URI"))
		// http.Redirect(w, r, oauthUrl, http.StatusFound)
		resp.Success(w, http.StatusOK, oauthUrl)
		return
	}

	redirectUri, redirectOK := parameters["redirect_uri"]

	_, pollingIDOK := parameters["generate_polling_id"]

	//generate a new base64url encoded signed with rsa encrypted state (random string) and stored on the db (plain)
	state, randomID, err := CreateState(redirectUri[0], redirectOK, pollingIDOK)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	//set the state on the session
	session.Values["state"] = state

	if pollingIDOK {
		session.Values["randomID"] = randomID
	}

	if err := session.Save(r, w); err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	oauthUrl := fmt.Sprintf("https://id.paleo.bg.it/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", os.Getenv("OAUTH_ID"), state, os.Getenv("REDIRECT_URI"))

	if pollingIDOK {
		response := map[string]string{"oauthUrl": oauthUrl, "randomID": randomID}
		resp.SuccessParse(w, http.StatusOK, "paleoid login url", response)
	} else {
		resp.Success(w, http.StatusOK, oauthUrl)
	}
}

func (h Handler) CheckOauthState(w http.ResponseWriter, r *http.Request) {
	pollingID, pollingIDexists := mux.Vars(r)["randomID"]
	if !pollingIDexists {
		resp.Error(w, http.StatusBadRequest, "no polling id found in the session")
		return
	}

	db, err := connectToDB()
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "unable to connect to database: %s", err.Error())
		return
	}

	id := make(map[string]interface{})
	pollingCollection := db.Collection("pollingIDs")
	err = pollingCollection.FindOne(context.Background(), bson.M{"id": pollingID}).Decode(&id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			resp.Error(w, http.StatusBadRequest, "polling id not found")
		} else {
			resp.Errorf(w, http.StatusInternalServerError, "error getting the id from the database: %s", err.Error())
		}
		return
	}

	if id["expDate"].(time.Time).Before(time.Now()) {
		_, err = pollingCollection.DeleteOne(context.TODO(), bson.M{"id": pollingID})
		if err != nil {
			resp.Errorf(w, http.StatusInternalServerError, "error deleting id from database: %s", err.Error())
		} else {
			resp.Error(w, http.StatusBadRequest, "id has expired")
		}
		return
	}

	response := make(map[string]interface{})
	if !id["loginSuccessful"].(bool) {
		response["loggedIn"] = false
		resp.ErrorParse(w, http.StatusBadRequest, "user not logged in yet", response)
		return
	}

	response["loggedIn"] = true
	response["accessToken"] = id["accessToken"]
	response["refreshToke"] = id["refreshToken"]
	_, err = pollingCollection.DeleteOne(context.TODO(), bson.M{"id": pollingID})
	if err != nil {
		resp.Errorf(w, http.StatusInternalServerError, "error deleting id from database: %s", err.Error())
		return
	}
	resp.SuccessParse(w, http.StatusBadRequest, "user logged in correctly, this token can't be used anymore now", response)
}

//get the user's informations from the ipaas access token
func (h Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectToDB()
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Client().Disconnect(context.TODO())

	log.Println("getting access token")
	//get the access token from the cookie
	cookie, err := r.Cookie("ipaas-access-token")
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
	cookie, err := r.Cookie("ipaas-refresh-token")
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
	defer db.Client().Disconnect(context.TODO())

	//check if the refresh token is expired
	isExpired, err := IsRefreshTokenExpired(refreshToken, db)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if isExpired {
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
		Name:    "ipaas-access-token",
		Path:    "/",
		Value:   "",
		Expires: time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "ipaas-refresh-token",
		Path:    "/",
		Value:   "",
		Expires: time.Unix(0, 0),
	})

	//set the new tokens
	//!should set domain and path
	http.SetCookie(w, &http.Cookie{
		Name:    "ipaas-access-token",
		Path:    "/",
		Value:   accessToken,
		Expires: time.Now().Add(time.Hour),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "ipaas-refresh-token",
		Path:    "/",
		Value:   newRefreshToken,
		Expires: time.Now().Add(time.Hour * 24 * 7),
	})

	//we also respond with the new tokens so the client doesn't have to depend from the cookies
	response := map[string]interface{}{
		"ipaas-access-token":  accessToken,
		"ipaas-refresh-token": newRefreshToken,
	}
	resp.SuccessParse(w, http.StatusOK, "New token pair generated", response)
}

//!===========================PAGES HANDLERS

//constructor
func NewHandler() (*Handler, error) {
	var h Handler
	var err error
	h.sess = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	h.cc, err = NewContainerController()
	if err != nil {
		return nil, err
	}
	h.util, err = NewUtil(h.cc.ctx)
	if err != nil {
		return nil, err
	}
	return &h, nil
}
