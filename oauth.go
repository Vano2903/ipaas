package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

//returns a unique signed base64url encoded state string that lasts 5 minutes (saved on the database)
func CreateState() (string, error) {
	//connect to the db
	db, err := connectToDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	//generate a random state string (must not already be on the db)
	var state string
	for {
		state = generateRandomString(24)
		var duplicate string
		err = db.QueryRow("SELECT state FROM states WHERE state = ?", state).Scan(&duplicate)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
		}
	}

	//save the state on the db (plain)
	_, err = db.Exec("INSERT INTO states (state) VALUES (?)", state)
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
	defer db.Close()

	//check if the state is actually found
	state := string(decryptedBytes)
	var issued time.Time
	err = db.QueryRow("SELECT issDate FROM states WHERE state = ?", state).Scan(&issued)
	if err != nil {
		return false, fmt.Errorf("state not found")
	}

	//delete the state from the database and check if it's still valid
	//(should delete it even if it's expired so we delete it before check if it's expired)
	db.Exec("DELETE FROM states WHERE state = ?", state)
	if time.Since(issued) > time.Minute*5 {
		return false, fmt.Errorf("state expired, it was issued %v ago", time.Since(issued))
	}

	return true, nil
}

//given the code generate from the paleoid server it returns the access token of the student
//this section is documented on the official paleoid documentation of how to retireve the access token
//https://paleoid.stoplight.io/docs/api/b3A6NDE0Njg2Mw-ottieni-un-access-token
func GetPaleoIDAccessToken(code string) (string, error) {
	url := "https://id.paleo.bg.it/oauth/token"
	payload := Payload{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectUri:  os.Getenv("REDIRECT_URI"),
		ClientID:     os.Getenv("OAUTH_ID"),
		ClientSecret: os.Getenv("OAUTH_SECRET"),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

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

	accessToken := string(body)
	accessToken = strings.Replace(accessToken, `{"access_token":"`, "", -1)
	accessToken = accessToken[:129]
	accessToken = strings.Replace(accessToken, "\n", "", -1)
	return accessToken, nil
}

//return a student struct given the access token
//this section is documented on the official paleoid documentation of
//how to retireve the student data from the access token
//https://paleoid.stoplight.io/docs/api/b3A6NDIwMTA1Mw-ottieni-le-informazioni-dell-utente
func GetStudent(accessToken string) (Student, error) {
	url := "https://id.paleo.bg.it/api/v2/user"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Student{}, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Student{}, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Student{}, err
	}

	var student Student
	err = json.Unmarshal(body, &student)
	return student, err
}
