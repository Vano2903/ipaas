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
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}

	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = &privateKey.PublicKey

	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
}
