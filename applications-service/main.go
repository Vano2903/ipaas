package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	service         = "applications-service"
	listeningQueue  = "send-applications"
	respondingQueue = "receive-applications"
)

var (
	MONGO_URI string
	AMPQ_URL  string
	Langs     []LangsStruct
)

type LangsStruct struct {
	Lang       string `bson:"lang"`
	Dockerfile string `bson:"dockerfile"`
	CanBeUsed  bool   `bson:"canBeUsed"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	if os.Getenv("LOG_TYPE") == "file" {
		log.SetFormatter(&log.JSONFormatter{})
		file, err := os.OpenFile(".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to log to file, using default stderr")
		}
		log.SetOutput(file)
	} else {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})
		log.SetOutput(os.Stdout)
	}

	log.SetLevel(log.WarnLevel)

	if os.Getenv("LOG_LEVEL") == "debug" {
		log.SetLevel(log.DebugLevel)
	} else if os.Getenv("LOG_LEVEL") == "info" {
		log.SetLevel(log.InfoLevel)
	}

	//checking if all envs are set
	MONGO_URI = os.Getenv("MONGO_URI")
	if MONGO_URI == "" {
		log.Fatal("MONGO_URI is not set in .env file")
	}
	AMPQ_URL = os.Getenv("AMPQ_URL")
	if AMPQ_URL == "" {
		log.Fatal("AMPQ_URL is not set in .env file")
	}

	//checking connection to database
	conn, err := ConnectToDB()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error connecting to database")
	}
	if err := conn.Client().Ping(context.Background(), readpref.Primary()); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error pinging the database")
	}
	defer conn.Client().Disconnect(context.Background())

	if err := LoadAvailableLangs(conn); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error loading available languages")
	}
	log.Debug("Available languages loaded")

	log.Info("Starting application service")
}

func LoadAvailableLangs(conn *mongo.Database) error {
	//declaring struct for languages
	cur, err := conn.Collection("langs").Find(context.TODO(), bson.D{})
	if err != nil {
		return err
	}

	return cur.All(context.TODO(), &Langs)
}

func main() {
	log.Debug("Connecting to RabbitMQ")
	conn, err := amqp.Dial(AMPQ_URL)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("error connecting to rabbitmq")
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("error creating channel")
	}
	defer ch.Close()

	log.Debugf("Declaring queue %s", listeningQueue)
	q, err := ch.QueueDeclare(
		listeningQueue, // name
		false,          // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to declare a queue")
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to consume messages")
	}

	var forever chan bool

	log.Debug("listening for messages...")
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Debug(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// func SetLangs(conn *mongo.Database) error {
// 	//declaring struct for languages
// 	lang := LangsStruct{
// 		Lang:       "go",
// 		Dockerfile: "golang:1.18.1-alpine3.15.dockerfile",
// 		CanBeUsed:  true,
// 	}
// 	_, err := conn.Collection("langs").InsertOne(context.TODO(), lang)
// 	return err
// }
