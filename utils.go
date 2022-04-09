package main

import (
	"database/sql"
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

type Util struct {
}

//get the student from the database given a valid access token (will be retrived from cookies)
func (u Util) GetUserFromCookie(r *http.Request, connection *sql.DB) (Student, error) {
	//search the access token in the cookies
	var acc string
	for _, cookie := range r.Cookies() {
		switch cookie.Name {
		case "accessToken":
			acc = cookie.Value
		}
	}

	//check if it's not empty
	if acc == "" {
		return Student{}, fmt.Errorf("no access token found")
	}

	//get the student from the database, it will automatically check if the access token is valid
	s, err := GetUserFromAccessToken(acc, connection)
	if err != nil {
		return Student{}, err
	}

	return s, nil
}

//generate a new pointer to the util struct
//is like a constructor
func NewUtil() (*Util, error) {
	return &Util{}, nil
}

//returns a pointer to a db connection
func connectToDB() (db *sql.DB, err error) {
	return sql.Open("mysql", "root:root@tcp(localhost:3306)/ipaas?parseTime=true&charset=utf8mb4")
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
