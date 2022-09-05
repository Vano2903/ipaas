package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

type CustomClaims struct {
	UserID int `json:"name,omitempty"`
	jwt.StandardClaims
}

type Parser struct {
	Secret []byte
}

func NewParser(secret []byte) *Parser {
	return &Parser{Secret: secret}
}

func (p Parser) GenerateToken(userID int, exp time.Duration) (string, error) {
	claims := CustomClaims{
		userID,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(exp).Unix(),
			Issuer:    "ipaas",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.Secret)
}

func (p Parser) ParseToken(t string) (CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		t,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return p.Secret, nil
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

func (p Parser) IsTokenExpired(t string) bool {
	claims, err := p.ParseToken(t)
	if err != nil {
		return true
	}
	return claims.ExpiresAt < time.Now().UTC().Unix()
}
