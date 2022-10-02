package main

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

type CustomClaims struct {
	UserID int `json:"name,omitempty"`
	jwt.StandardClaims
}

func NewCustomClaims(userID int) CustomClaims {
	token := CustomClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			Issuer:    "ipaas-backend",
		},
	}
	return token
}

func NewSignedToken(claim CustomClaims) (string, error) {
	//unsigned token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	//sign the token
	return token.SignedString(JwtSecret)
}

func ParseToken(t string) (CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		t,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		},
	)
	if err != nil {
		return CustomClaims{}, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return CustomClaims{}, errors.New("can't parse claims")
	}

	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return CustomClaims{}, errors.New("jwt is expired")
	}
	return *claims, nil
}

func IsJWTexpired(t string) bool {
	claims, err := ParseToken(t)
	if err != nil {
		return true
	}
	return claims.ExpiresAt < time.Now().UTC().Unix()
}
