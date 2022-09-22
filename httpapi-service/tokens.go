package main

import (
	"context"
	"fmt"
	"github.com/vano2903/ipaas/internal/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RefreshToken struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     int                `bson:"userID"`
	Token      string             `bson:"token"`
	Expiration time.Time          `bson:"expiration"`
}

// check if a refresh token is already in the database
func DoesRefreshTokenAlreadyExists(token string, connection *mongo.Database) (bool, error) {
	RefreshTokensCollection := connection.Collection("refreshTokens")
	var result []RefreshToken
	err := RefreshTokensCollection.
		FindOne(context.TODO(), bson.M{"token": token}).
		Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// generate a new token pair from the userID (matricola)
func GenerateTokenPair(userID int, connection *mongo.Database) (string, string, error) {
	//generate the access token
	jwtAccessToken, err := parser.GenerateToken(userID, time.Minute*15)
	if err != nil {
		return "", "", err
	}

	token := RefreshToken{}
	token.UserID = userID
	//generate a random string for the refresh token
	//and set the expiration time to 1 week
	for {
		token.Token = utils.GenerateRandomString(64)
		token.Expiration = time.Now().Add(time.Hour * 24 * 7)
		//check if the access token is already in the database
		found, err := DoesRefreshTokenAlreadyExists(token.Token, connection)
		if err != nil {
			return "", "", err
		}

		if !found {
			break
		}
	}

	_, err = connection.Collection("refreshTokens").InsertOne(context.TODO(), token)
	if err != nil {
		return "", "", err
	}

	return jwtAccessToken, token.Token, nil
}

// check if the refresh token is expired
func IsRefreshTokenExpired(token string, connection *mongo.Database) (bool, error) {
	RefreshTokensCollection := connection.Collection("refreshTokens")

	var result RefreshToken
	err := RefreshTokensCollection.
		FindOne(context.TODO(), bson.M{"token": token}).
		Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return true, nil
		}
		return true, err
	}

	//check if the expiration time is "before" the current time
	return result.Expiration.Before(time.Now()), nil
}

// get the student struct from the access token (can't use the refresh token)
func GetUserFromAccessToken(accessToken string, connection *mongo.Database) (Student, error) {
	claims, err := parser.ParseToken(accessToken)
	if err != nil {
		return Student{}, err
	}
	var user Student
	err = connection.Collection("users").
		FindOne(context.TODO(), bson.M{"userID": claims.UserID}).
		Decode(&user)
	return user, err
}

// generate a new token pair given a valid refresh token
// the refresh token allows us to get the userID that will be used to generat a new token pair
func GenerateNewTokenPairFromRefreshToken(refreshToken string, connection *mongo.Database) (string, string, error) {
	//check if the refresh token is expired
	isExpired, err := IsRefreshTokenExpired(refreshToken, connection)
	if err != nil {
		return "", "", err
	}

	if isExpired {
		return "", "", fmt.Errorf("refresh token is expired")
	}

	//get the user id from the refresh token
	var token RefreshToken
	tokensCollection := connection.Collection("refreshTokens")
	err = tokensCollection.
		FindOne(context.TODO(), bson.M{"token": refreshToken}).
		Decode(&token)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", "", fmt.Errorf("no refresh token found")
		}
		return "", "", err
	}

	_, err = tokensCollection.DeleteOne(context.TODO(), bson.M{"token": refreshToken})
	if err != nil {
		return "", "", err
	}

	//generate the new token pair
	return GenerateTokenPair(token.UserID, connection)
}
