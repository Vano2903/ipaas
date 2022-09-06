package main

import (
	"context"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

//func (m MessageBody) Validate(action string ) error

type Message struct {
	UserID          int       `json:"userId"`
	UserAccessToken string    `json:"userAccessToken"`
	Action          string    `json:"action"`
	Issued          time.Time `json:"issued"`
	MessageID       string    `json:"messageId"`
	Body            AppPost   `json:"body"`
}

func (m Message) IsValidAction(performer Student) bool {
	action, ok := Actions[m.Action]
	if !ok {
		return false
	}

	if !action.CanBePerformed {
		return false
	}

	//admins can't be blacklisted
	if action.AdminOnly && performer.IsAdmin {
		return true
	}

	//check if the user is blacklisted
	for _, blacklisted := range action.Blacklist {
		if blacklisted == performer.ID {
			return false
		}
	}

	return true
}

func (m Message) Validate() (Student, error) {
	db, err := u.ConnectToDB()
	if err != nil {
		return Student{}, err
	}

	defer func(client *mongo.Client, ctx context.Context) (Student, error) {
		err := client.Disconnect(ctx)
		if err != nil {
			return Student{}, err
		}
		return Student{}, nil
	}(db.Client(), context.Background())

	//check if accessToken is valid and matches with the userID
	claim, err := parser.ParseToken(m.UserAccessToken)
	if err != nil {
		return Student{}, err
	}

	if claim.UserID != m.UserID {
		return Student{}, errors.New("userID and access token do not match")
	}

	//check if the userID exists in the database
	var found Student
	err = db.Collection("users").FindOne(context.TODO(), bson.M{"userID": m.UserID}).Decode(&found)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Student{}, errors.New("user not found")
		} else {
			return Student{}, err
		}
	}

	//check if the action is a valid action
	if !m.IsValidAction(found) {
		return Student{}, errors.New("action can't be performed")
	}

	return found, nil
}

func MessageHandler(msg amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Info("Received message")

	var message Message
	err := json.Unmarshal(msg.Body, &message)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("error validating message")
		return
	}

	_, err = message.Validate()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("error validating message")
		return
	}

	switch message.Action {
	case "createApp":
		//create app
		app, err := cont.CreateNewApplication(message.UserID, message.Body)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err,
			}).Error("error creating app")
			//msg.Nack(false, false)
			return
		}
		logger.WithFields(log.Fields{
			"userID":      message.UserID,
			"containerID": app.ID,
		}).Info("app created")
	case "deleteApp":
		//delete app
	case "updateApp":
		//update app
	}

	logger.WithFields(log.Fields{
		"message": message,
	}).Info("Message handled")
}
