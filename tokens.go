package main

import (
	"database/sql"
	"fmt"
	"time"
)

type TokenPair struct {
	AccessToken            string
	ExpirationAccessToken  time.Time
	RefreshToken           string
	ExpirationRefreshToken time.Time
}

//check if a token is already in the database (either access or refresh token)
func tokenExists(token string, connection *sql.DB) (bool, error) {
	query := "SELECT * FROM tokens WHERE accToken = ? OR refreshToken = ?"

	var found bool
	err := connection.QueryRow(query, token, token).Scan(&found)
	if err != nil {
		//if the error is ErrNoRows then the token is not in the database
		if err == sql.ErrNoRows {
			fmt.Printf("token %s not found", token)
			return false, nil
		}
		return false, err
	}

	return true, nil
}

//generate a new token pair from the userID (matricola)
func GenerateTokenPair(userID int, connection *sql.DB) (string, string, error) {
	//generate a random string for the access token
	pair := TokenPair{}

	//generate a random 64 character string and check that's not in the db
	//generate the access token and it expires in 1 hour
	for {
		pair.AccessToken = generateRandomString(64)
		pair.ExpirationAccessToken = time.Now().Add(time.Minute)
		//check if the access token is already in the database
		found, err := tokenExists(pair.AccessToken, connection)
		if err != nil {
			return "", "", err
		}

		if !found {
			break
		}
	}

	//same thing of the access token but for the refresh token, the expiration time is longer (7 days)
	for {
		pair.RefreshToken = generateRandomString(64)
		pair.ExpirationRefreshToken = time.Now().Add(time.Hour * 24 * 7)
		//check if the access token is already in the database
		found, err := tokenExists(pair.RefreshToken, connection)
		if err != nil {
			return "", "", err
		}

		if !found && pair.AccessToken != pair.RefreshToken {
			break
		}
	}

	//insert the token pair into the database
	query := "INSERT INTO tokens (userID, accToken, accExp, refreshToken, refreshExp) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE accToken = ?, accExp = ?, refreshToken = ?, refreshExp = ?"
	_, err := connection.Exec(query, userID, pair.AccessToken, pair.ExpirationAccessToken, pair.RefreshToken, pair.ExpirationRefreshToken, pair.AccessToken, pair.ExpirationAccessToken, pair.RefreshToken, pair.ExpirationRefreshToken)

	return pair.AccessToken, pair.RefreshToken, err
}

//check if the token is expired (must define the type of token)
func IsTokenExpired(isAccessToken bool, token string, connection *sql.DB) bool {
	var query string
	if isAccessToken {
		query = "SELECT accExp FROM tokens WHERE accToken = ?"
	} else {
		query = "SELECT refreshExp FROM tokens WHERE refreshToken = ?"
	}

	var exp time.Time
	err := connection.QueryRow(query, token).Scan(&exp)
	if err != nil {
		return true
	}

	//check if the expiration time is "before" the current time
	return exp.Before(time.Now())
}

//che the student struct from the access token (can't use the refresh token)
func GetUserFromAccessToken(accessToken string, connection *sql.DB) (Student, error) {
	//check if the token is expired
	if IsTokenExpired(true, accessToken, connection) {
		return Student{}, nil
	}

	query := "SELECT u.* FROM users as u JOIN tokens t ON u.userID = t.userID WHERE t.accToken = ?"
	var user Student
	//get the student usign a join between the tokens and the users
	err := connection.QueryRow(query, accessToken).Scan(&user.ID, &user.Name, &user.LastName, &user.Email, &user.Pfp)
	return user, err
}

//generate a new token pair given a valid refresh token
//the refresh token allows us to get the userID that will be used to generat a new token pair
func GenerateNewTokenPairFromRefreshToken(refreshToken string, connection *sql.DB) (string, string, error) {
	//check if the refresh token is expired
	if IsTokenExpired(false, refreshToken, connection) {
		return "", "", nil
	}
	//query to get the userID from the refresh token
	getUserFromRefreshTokenQuery := "SELECT u.userID FROM users as u JOIN tokens t ON u.userID = t.userID WHERE t.refreshToken = ?"
	var userID int
	err := connection.QueryRow(getUserFromRefreshTokenQuery, refreshToken).Scan(&userID)
	if err != nil {
		return "", "", err
	}
	//generate the new token pair
	return GenerateTokenPair(userID, connection)
}
