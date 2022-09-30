package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	handler    *Handler
	Langs      []string
)

func init() {
	//load the enviroment variables
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}

	DATABASE_URI = os.Getenv("DB_URI")
	fmt.Println(DATABASE_URI)
	JWT_SECRET = []byte(os.Getenv("JWT_SECRET"))

	conn, err := connectToDB()
	if err != nil {
		panic("error connecting to the database: " + err.Error())
	}
	if err := conn.Client().Ping(context.TODO(), readpref.Primary()); err != nil {
		panic("error pinging the database: " + err.Error())
	}
	defer conn.Client().Disconnect(context.TODO())

	if err := initDatabase(conn); err != nil {
		panic("error initializing the database: " + err.Error())
	}

	//generate private and public keys
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = &privateKey.PublicKey

	var LangsStruct []struct {
		Lang string
	}
	cur, err := conn.Collection("langs").Find(context.TODO(), bson.D{})
	if err != nil {
		panic("error getting supported langs " + err.Error())
	}

	err = cur.All(context.TODO(), &LangsStruct)
	if err != nil {
		panic("error reading supported langs " + err.Error())
	}

	for _, Lang := range LangsStruct {
		Langs = append(Langs, Lang.Lang)
	}

	handler, err = NewHandler()
	if err != nil {
		panic(err)
	}
}
