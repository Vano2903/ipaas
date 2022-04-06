package main

import (
	"database/sql"
	"math/rand"
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
