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

type Student struct {
	Name        string      `json:"nome"`
	LastName    string      `json:"cognome"`
	Email       string      `json:"email"`
	Type        string      `json:"tipo"`
	ID          int         `json:"matricola"`
	StudentInfo StudentInfo `json:"info_studente"`
}

type StudentInfo struct {
	Class               string `json:"classe"`
	Year                int    `json:"anno"`
	Field               string `json:"indirizzo"`
	IsClassPresident    bool   `json:"rappresentante_classe"`
	IsIstiturePresident bool   `json:"rappresentante_istituto"`
}

//returns a unique signed base64url encoded state string (saved on the database)
func CreateState() (string, error) {
	db, err := connectToDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var state string
	for {
		state = generatePassword()
		var duplicate string
		err = db.QueryRow("SELECT state FROM states WHERE state = ?", state).Scan(&duplicate)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
		}
	}

	_, err = db.Exec("INSERT INTO states (state) VALUES (?)", state)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		[]byte(state),
		nil)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

//check if the encrypted state is valid and if so returnes true and delete the state from the database
func CheckState(cypher string) (bool, error) {
	cypher = strings.Replace(cypher, " ", "+", -1)
	decoded, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return false, err
	}
	decryptedBytes, err := privateKey.Decrypt(nil, decoded, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		return false, err
	}

	db, err := connectToDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	state := string(decryptedBytes)
	var issued time.Time
	err = db.QueryRow("SELECT issDate FROM states WHERE state = ?", state).Scan(&issued)
	if err != nil {
		return false, fmt.Errorf("state not found")
	}

	db.Exec("DELETE FROM states WHERE state = ?", state)
	if time.Since(issued) > time.Minute*5 {
		return false, fmt.Errorf("state expired, it was issued %v ago", time.Since(issued))
	}

	return true, nil
}

//given the code it returns the access token of the student
func GetAccessToken(code string) (string, error) {
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
