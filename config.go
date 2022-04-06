package main

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/joho/godotenv"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	handler    *Handler
)

func init() {
	//load the enviroment variables
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}

	conn, err := connectToDB()
	if err != nil {
		panic("error connecting to the database: " + err.Error())
	}
	err = conn.Ping()
	if err != nil {
		panic("error pinging the database: " + err.Error())
	}

	//generate private and public keys
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = &privateKey.PublicKey

	handler, err = NewHandler()
	if err != nil {
		panic(err)
	}
}
