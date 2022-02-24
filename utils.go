package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

func connectToDB() (db *sql.DB, err error) {
	return sql.Open("mysql", "root:root@tcp(localhost:3306)/ipaas?parseTime=true&charset=utf8mb4")
}

func returnError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true}`, code, message)
}

func returnErrorJson(w http.ResponseWriter, code int, message, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func returnSuccess(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false}`, code, message)
}

func returnSuccessJson(w http.ResponseWriter, code int, message string, values map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, _ := json.Marshal(values)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}

// func checkJWT(w http.ResponseWriter, r *http.Request) (CustomClaims, error) {
// 	jwt, err := r.Cookie("JWT")
// 	if err != nil {
// 		returnError(w, http.StatusUnauthorized, "No JWT cookie found")
// 		return CustomClaims{}, err
// 	}

// 	jwtContent, err := ParseToken(jwt.Value)
// 	if err != nil {
// 		returnError(w, http.StatusUnauthorized, "Invalid JWT")
// 		return CustomClaims{}, err
// 	}
// 	return jwtContent, nil
// }
