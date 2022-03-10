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

//returns a pointer to a db connection
func connectToDB() (db *sql.DB, err error) {
	return sql.Open("mysql", "root:root@tcp(localhost:3306)/ipaas?parseTime=true&charset=utf8mb4")
}

func returnError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true}`, code, message)
}

func returnErrorMap(w http.ResponseWriter, code int, message string, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func returnErrorJson(w http.ResponseWriter, code int, message string, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func returnSuccess(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false}`, code, message)
}

func returnSuccessMap(w http.ResponseWriter, code int, message string, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}

func returnSuccessJson(w http.ResponseWriter, code int, message string, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}

//function to generate a random alphanumerical string without spaces and with a given length
func generateRandomString(size int) string {
	minNum := 4
	minUpperCase := 4
	passwordLength := size

	rand.Seed(time.Now().UnixNano())
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