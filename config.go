package main

import (
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env.yaml")
	if err != nil {
		panic("Error loading .env file")
	}
}
