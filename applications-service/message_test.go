package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/vano2903/ipaas/internal/jwt"
)

type Test struct {
	Message   Message
	ShouldErr bool
}

func TestValidate(t *testing.T) {
	jwtP := jwt.NewParser([]byte(os.Getenv("JWT_SECRET")))
	validAccessToken, err := jwtP.GenerateToken(18008, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	toTest := []Test{
		{Message{0, "", "", time.Now(), "1", AppPost{}}, true},
		{Message{18008, "invalid accessToken", "", time.Now(), "1", AppPost{}}, true},
		{Message{18008, "valid accessToken", "", time.Now(), "1", AppPost{}}, true},
		{Message{18008, validAccessToken, "invalid", time.Now(), "1", AppPost{}}, true},
		{Message{18008, validAccessToken, "createApp", time.Now(), "1", AppPost{}}, false},
	}

	fmt.Println(Actions)

	for _, test := range toTest {
		s, err := test.Message.Validate()
		if !test.ShouldErr {
			if err != nil {
				t.Errorf("Should not have errored, but got error: %v", err)
			}
		}
		fmt.Println("student", s)
	}
}
