package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// CheckExpiries checks the expiries of the given collection and deletes the expired documents
func CheckExpiries(execute chan string) {
	//execute the cleaning of a collection when the channel is triggered

	db, err := connectToDB()
	if err != nil {
		log.Printf("[ERROR] Error connecting to database: %v\n", err)
		return
	}
	defer db.Client().Disconnect(context.Background())

	oauthStatesCollection := db.Collection("oauthStates")
	refreshTokensCollection := db.Collection("refreshTokens")
	usersTokensCollections := db.Collection("users")
	pollingIDsCollection := db.Collection("pollingIDs")

	for {
		collection := <-execute
		switch collection {
		case "oauthStates":
			log.Println("[DEBUG] Cleaning oauthStates collection")

			stateCur, err := oauthStatesCollection.Find(context.Background(), bson.M{})
			if err != nil {
				log.Printf("[ERROR] Error getting the oauth states: %v\n", err)
				continue
			}
			var states []State
			err = stateCur.All(context.TODO(), &states)
			if err != nil {
				log.Println("[ERROR] Error getting all the states")
				continue
			}
			for _, s := range states {
				if s.ExpirationDate.Before(time.Now()) {
					_, err = oauthStatesCollection.DeleteOne(context.Background(), bson.M{"_id": s.Id})
					if err != nil {
						log.Printf("[ERROR] Error deleting the state %s: %v\n", s.State, err)
						continue
					}
					log.Printf("[INFO] Deleted state %s\n", s.State)
				}
			}
			log.Println("[DEBUG] oauthStates collection cleaned")
		case "refreshTokens":
			log.Println("[DEBUG] Cleaning refreshTokens collection")

			refreshTokensCur, err := refreshTokensCollection.Find(context.Background(), bson.M{})
			if err != nil {
				log.Printf("[ERROR] Error getting the refresh tokens: %v\n", err)
				continue
			}
			var refreshTokens []RefreshToken
			err = refreshTokensCur.All(context.TODO(), &refreshTokens)
			if err != nil {
				log.Println("[ERROR] Error getting all the refresh tokens")
				continue
			}
			for _, r := range refreshTokens {
				if r.Expiration.Before(time.Now()) {
					_, err = refreshTokensCollection.DeleteOne(context.Background(), bson.M{"_id": r.ID})
					if err != nil {
						log.Printf("[ERROR] Error deleting the refresh token (%s) for %d: %v\n", r.Token, r.UserID, err)
						continue
					}
					log.Printf("[INFO] Deleted refresh token (%s) for %d\n", r.Token, r.UserID)
				}
			}
			log.Println("[DEBUG] refreshTokens collection cleaned")
		case "users":
			log.Println("[DEBUG] Cleaning users collection")

			usersCur, err := usersTokensCollections.Find(context.Background(), bson.M{"isMock": true})
			if err != nil {
				log.Printf("[ERROR] Error getting the users: %v\n", err)
				continue
			}
			var users []struct {
				UserID       int       `bson:"userID"`
				Password     string    `bson:"password"`
				Name         string    `bson:"name"`
				Pfp          string    `bson:"pfp"`
				CreationDate time.Time `bson:"creationDate"`
				IsMock       bool      `bson:"isMock"`
			}
			err = usersCur.All(context.Background(), &users)
			if err != nil {
				log.Println("[ERROR] Error getting all the users")
				continue
			}
			for _, u := range users {
				if u.CreationDate.AddDate(0, 0, 1).Before(time.Now()) {
					_, err = usersTokensCollections.DeleteOne(context.Background(), bson.M{"userID": u.UserID})
					if err != nil {
						log.Printf("[ERROR] Error deleting the user (%d): %v\n", u.UserID, err)
						continue
					}
					log.Printf("[INFO] Deleted mock user %s [%d] \n", u.Name, u.UserID)
				}
			}
			log.Println("[DEBUG] users collection cleaned")
		case "pollingIDs":
			log.Println("[DEBUG] Cleaning pollingIDs collection")

			pollingIDsCur, err := pollingIDsCollection.Find(context.Background(), bson.M{})
			if err != nil {
				log.Printf("[ERROR] Error getting the polling IDs: %v\n", err)
				continue
			}
			var pollingIDs []Polling
			err = pollingIDsCur.All(context.Background(), &pollingIDs)
			if err != nil {
				log.Printf("[ERROR] Error getting the polling IDs: %v\n", err)
				continue
			}
			for _, p := range pollingIDs {
				if p.ExpDate.Before(time.Now()) {
					_, err = pollingIDsCollection.DeleteOne(context.Background(), bson.M{"_id": p.DBId})
					if err != nil {
						log.Printf("[ERROR] Error deleting the polling ID (%s): %v\n", p.DBId.String(), err)
						continue
					}
					success := "failed"
					if p.LoginSuccessful {
						success = "success"
					}
					log.Printf("[INFO] Deleted polling ID %s of a %s login", p.RandomId, success)
				}
			}
			log.Println("[DEBUG] pollingIDs collection cleaned")
		}
	}
}

// RunExecutor runs CheckExpiries function every given interval sending the event on the channel
func RunExecutor(oauthStateCleaningInterval, refreshTokensCleaningInterval, usersCleaningInterval, pollingIDsCleaningInterval time.Duration, execute chan string) {
	go func() {
		for {
			execute <- "oauthStates"
			time.Sleep(oauthStateCleaningInterval)
		}
	}()

	go func() {
		for {
			execute <- "refreshTokens"
			time.Sleep(refreshTokensCleaningInterval)
		}
	}()

	go func() {
		for {
			execute <- "users"
			time.Sleep(usersCleaningInterval)
		}
	}()

	go func() {
		for {
			execute <- "pollingIDs"
			time.Sleep(pollingIDsCleaningInterval)
		}
	}()
}
