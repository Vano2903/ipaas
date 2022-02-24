package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
)

//returns a unique signed base64url encoded state string (saved on the database)
func CreateState() (string, error) {
	db, err := connectToDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var state string
	for {
		state = generatePassword()
		var duplicate string
		err = db.QueryRow("SELECT state FROM states WHERE state = ?", state).Scan(&duplicate)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
		}
	}

	res, err := db.Exec("INSERT INTO states (state) VALUES (?)", state)
	if err != nil {
		return "", err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	fmt.Println("stato id: ", id)

	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		[]byte(state),
		nil)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

func CheckState(cypher string) (bool, error) {
	decoded, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return false, err
	}
	decryptedBytes, err := privateKey.Decrypt(nil, decoded, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		return false, err
	}

	db, err := connectToDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	state := string(decryptedBytes)
	var issued time.Time
	err = db.QueryRow("SELECT issDate FROM states WHERE state = ?", state).Scan(&issued)
	if err != nil {
		return false, fmt.Errorf("state not found")
	}

	db.Exec("DELETE FROM states WHERE state = ?", state)
	if time.Now().Sub(issued) > time.Minute*5 {
		return false, fmt.Errorf("state expired, it was issued %v ago", time.Now().Sub(issued))
	}

	return true, nil
}
