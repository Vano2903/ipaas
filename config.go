package main

import (
	"crypto/rand"
	"crypto/rsa"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	store      *sessions.CookieStore
)

func init() {
	//load the enviroment variables
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}

	//generate private and public keys
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = &privateKey.PublicKey

	//generate the session storage
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
}
