package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// struct used to get the user from paleoid
type Payload struct {
	GrantType    string `json:"grant_type"`    //will always be "authorization_code"
	Code         string `json:"code"`          //the code returned by the oauth server
	RedirectUri  string `json:"redirect_uri"`  //the redirect uri (saved in env variable)
	ClientID     string `json:"client_id"`     //the client id (saved in env variable)
	ClientSecret string `json:"client_secret"` //the client secret (saved in env variable)
}

type State struct {
	Id             primitive.ObjectID `bson:"_id"`
	State          string             `json:"state"`
	Issued         time.Time          `bson:"issDate"`
	ExpirationDate time.Time          `bson:"expDate"`
}

//returns a unique signed base64url encoded state string that lasts 5 minutes (saved on the database)
func CreateState() (string, error) {
	//connect to the db
	db, err := connectToDB()
	if err != nil {
		return "", err
	}
	defer db.Client().Disconnect(context.TODO())

	statesCollection := db.Collection("oauthStates")
	//generate a random state string (must not already be on the db)
	var state string
	for {
		state = generateRandomString(24)
		var duplicate string
		err = statesCollection.FindOne(context.TODO(), bson.M{"state": state}).Decode(&duplicate)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break
			}
		}
	}

	_, err = statesCollection.InsertOne(context.TODO(), bson.M{"state": state, "issDate": time.Now(), "expDate": time.Now().Add(time.Minute * 5)})
	if err != nil {
		return "", err
	}
	//encrypt the state
	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		[]byte(state),
		nil)
	if err != nil {
		return "", err
	}

	//encode the encrypted state with base64url
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

//check if the encrypted state is valid and if so returnes true and delete the state from the database
func CheckState(cypher string) (bool, error) {
	//replace the spaces with + signs in the cypher
	cypher = strings.Replace(cypher, " ", "+", -1)
	//decode the cypher with base64url
	decoded, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return false, err
	}

	//decrypt the cypher with the private key
	decryptedBytes, err := privateKey.Decrypt(nil, decoded, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		return false, err
	}

	db, err := connectToDB()
	if err != nil {
		return false, err
	}
	defer db.Client().Disconnect(context.TODO())

	//check if the state is actually found
	state := string(decryptedBytes)

	fmt.Println(state)

	stateCollection := db.Collection("oauthStates")

	var s State
	err = stateCollection.FindOne(context.TODO(), bson.M{"state": state}).Decode(&s)
	fmt.Println(s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("ahhhh non ho trovato nulla :(")
			return false, fmt.Errorf("state not found")

		}
		return false, err
	}

	//delete the state from the database and check if it's still valid
	//(should delete it even if it's expired so we delete it before check if it's expired)
	_, err = stateCollection.DeleteOne(context.TODO(), bson.M{"state": state})
	if err != nil {
		return false, err
	}

	if time.Since(s.Issued) > time.Minute*5 {
		return false, fmt.Errorf("state expired, it was issued %v ago", time.Since(s.Issued))
	}

	return true, nil
}

//given the code generate from the paleoid server it returns the access token of the student
//this section is documented on the official paleoid documentation of how to retireve the access token
//https://paleoid.stoplight.io/docs/api/b3A6NDE0Njg2Mw-ottieni-un-access-token
func GetPaleoIDAccessToken(code string) (string, error) {
	//do post request to url with the code and the env variables
	//(they are envs cause they are private and saved in the .env)
	url := "https://id.paleo.bg.it/oauth/token"
	payload := Payload{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectUri:  os.Getenv("REDIRECT_URI"),
		ClientID:     os.Getenv("OAUTH_ID"),
		ClientSecret: os.Getenv("OAUTH_SECRET"),
	}

	//encode the payload
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	//do the push request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//get the access token
	//I decided to use strings replace because it's easier and doesn't require a struct to unmarshal
	accessToken := string(body)
	accessToken = strings.Replace(accessToken, `{"access_token":"`, "", -1)
	//length of the paleoid access token
	accessToken = accessToken[:129]
	accessToken = strings.Replace(accessToken, "\n", "", -1)
	return accessToken, nil
}

//return a student struct given the access token
//this section is documented on the official paleoid documentation of
//how to retireve the student data from the access token
//https://paleoid.stoplight.io/docs/api/b3A6NDIwMTA1Mw-ottieni-le-informazioni-dell-utente
func GetStudentFromPaleoIDAccessToken(accessToken string) (Student, error) {
	url := "https://id.paleo.bg.it/api/v2/user"

	//make a get request to url with the access token as Bearer token
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Student{}, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	//make the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Student{}, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Student{}, err
	}

	//parse the body into a student struct (from the json response)
	var student Student
	err = json.Unmarshal(body, &student)
	return student, err
}
