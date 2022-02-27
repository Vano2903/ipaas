package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
)

func connectToDB() (db *sql.DB, err error) {
	return sql.Open("mysql", "root:root@tcp(localhost:3306)/ipaas?parseTime=true&charset=utf8mb4")
}

func returnError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true}`, code, message)
}

func returnErrorJson(w http.ResponseWriter, code int, message, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func returnSuccess(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false}`, code, message)
}

func returnSuccessJson(w http.ResponseWriter, code int, message string, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}

//function to generate a random alphanumerical password without spaces (24 characters)
func generatePassword() string {
	minNum := 4
	minUpperCase := 4
	passwordLength := 16

	rand.Seed(time.Now().Unix())
	var password strings.Builder

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

// func checkJWT(w http.ResponseWriter, r *http.Request) (CustomClaims, error) {
// 	jwt, err := r.Cookie("JWT")
// 	if err != nil {
// 		returnError(w, http.StatusUnauthorized, "No JWT cookie found")
// 		return CustomClaims{}, err
// 	}

// 	jwtContent, err := ParseToken(jwt.Value)
// 	if err != nil {
// 		returnError(w, http.StatusUnauthorized, "Invalid JWT")
// 		return CustomClaims{}, err
// 	}
// 	return jwtContent, nil
// }
