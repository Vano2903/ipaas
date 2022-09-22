package main

import (
	"context"
	"log"
	"net/http"

	resp "github.com/vano2903/ipaas/responser"
)

// check if the user has a valid access Token
func (h Handler) TokensMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("tokens middleware")

		// get the access token from cookies
		var accessToken string
		for _, cookie := range r.Cookies() {
			switch cookie.Name {
			case "ipaas-access-token":
				accessToken = cookie.Value
			}
		}

		//check if it's not empty
		//498 => token invalid/expired
		if accessToken == "" {
			resp.Error(w, 498, "No access token")
			return
		}

		//create a connection with the db
		db, err := u.ConnectToDB()
		if err != nil {
			resp.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer db.Client().Disconnect(context.TODO())

		//check if it's expired
		if parser.IsTokenExpired(accessToken) {
			resp.Error(w, 498, "Access token is expired")
			return
		}

		//redirect to the actual handler
		next.ServeHTTP(w, r)
	})
}
