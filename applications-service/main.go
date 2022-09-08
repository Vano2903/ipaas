package main

import (
	"context"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/vano2903/ipaas/internal/jwt"
	"github.com/vano2903/ipaas/internal/messanger"
	"github.com/vano2903/ipaas/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	service         = "applications-service"
	listeningQueue  = service + "-listening"
	respondingQueue = service + "-responses"
)

var (
	AmpqUrl       string
	MongoUri      string
	MaxGoroutines = 2
	Langs         []LangsStruct
	cont          *ContainerController
	parser        *jwt.Parser
	u             *utils.Util
	mess          *messanger.Messanger
	l             *log.Entry
	Actions       = make(map[string]Action)
)

type LangsStruct struct {
	Lang       string `bson:"lang"`
	Dockerfile string `bson:"dockerfile"`
	CanBeUsed  bool   `bson:"canBeUsed"`
}

type Action struct {
	Service        string `bson:"service"`
	Name           string `bson:"name" json:"name"`
	AdminOnly      bool   `bson:"adminOnly" json:"adminOnly"`
	CanBePerformed bool   `bson:"canBePerformed" json:"canBePerformed"`
	Blacklist      []int  `bson:"blacklist" json:"blacklist"`
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

	l = log.WithFields(log.Fields{
		"service": service,
	})

	//checking if all envs are set
	MongoUri = os.Getenv("MONGO_URI")
	if MongoUri == "" {
		l.Fatal("MONGO_URI is not set in .env file")
	}
	u = utils.NewUtil(context.TODO(), MongoUri)

	AmpqUrl = os.Getenv("AMPQ_URL")
	if AmpqUrl == "" {
		log.Fatal("AMPQ_URL is not set in .env file")
	}

	//checking connection to database
	conn, err := u.ConnectToDB()
	if err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error connecting to database")
	}
	if err := conn.Client().Ping(context.Background(), readpref.Primary()); err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error pinging the database")
	}
	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			l.WithFields(log.Fields{
				"error": err,
			}).Fatal("Error disconnecting from database")
		}
	}(conn.Client(), context.Background())

	//if err := SetLangs(conn); err != nil {
	//	log.WithFields(log.Fields{
	//		"error": err,
	//	}).Fatal("Error setting languages")
	//}

	if err := LoadAvailableLangs(conn); err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error loading available languages")
	}
	l.Debug("Available languages loaded")

	if err := LoadActionsFromDatabase(conn); err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error loading actions from database")
	}
	l.Debugln("Actions loaded")

	if os.Getenv("MAX_GOROUTINES") != "" {
		MaxGoroutines, err = strconv.Atoi(os.Getenv("MAX_GOROUTINES"))
		if err != nil {
			l.WithFields(log.Fields{
				"error": err,
			}).Fatal("Error converting MAX_GOROUTINES to int")
		}
		if MaxGoroutines <= 0 {
			l.Fatal("MAX_GOROUTINES must be greater than 0")
		}
	}

	parser = jwt.NewParser([]byte(os.Getenv("JWT_SECRET")))
	cont, err = NewContainerController(context.Background())
	mess = messanger.NewMessanger(AmpqUrl, listeningQueue, respondingQueue)

	l.Info("Starting application service")
}

func LoadAvailableLangs(conn *mongo.Database) error {
	//declaring struct for languages
	cur, err := conn.Collection("langs").Find(context.TODO(), bson.D{})
	if err != nil {
		return err
	}

	return cur.All(context.TODO(), &Langs)
}

func LoadActionsFromDatabase(conn *mongo.Database) error {
	cur, err := conn.Collection("actions").Find(context.Background(), bson.M{"service": service})
	if err != nil {
		return err
	}

	for cur.Next(context.Background()) {
		var action Action
		err := cur.Decode(&action)
		if err != nil {
			return err
		}
		Actions[action.Name] = action
	}
	return nil
}

func main() {
	l.Debug("Connecting to RabbitMQ")
	if err := mess.Connect(); err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to connect to rabbitmq")
	}

	msgs, err := mess.Listen()
	if err != nil {
		l.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to listen to rabbitmq")
	}

	defer func() {
		if err := mess.Close(); err != nil {
			l.WithFields(log.Fields{
				"error": err,
			}).Fatal("failed to disconnect from rabbitmq")
		}
	}()

	var wg sync.WaitGroup
	var routinesLimit int
	forever := make(chan bool)
	go func() {
		l.Debug("listening for messages...")
		for d := range msgs {
			wg.Add(1)
			routinesLimit++
			go MessageHandler(d, &wg)
			if routinesLimit >= MaxGoroutines {
				wg.Wait()
				routinesLimit = 0
			}
		}
	}()

	l.Debug(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	//TODO: should implement a graceful shutdown
}

//func SetLangs(conn *mongo.Database) error {
//	//declaring struct for languages
//	lang := LangsStruct{
//		Lang:       "go",
//		Dockerfile: "golang:1.18.1-alpine3.15.dockerfile",
//		CanBeUsed:  true,
//	}
//	_, err := conn.Collection("langs").InsertOne(context.TODO(), lang)
//	return err
//}

//func SetAction(conn *mongo.Database) error {
//	//declaring struct for languages
//	actions := []Action{{
//		Service:        service,
//		Name:           "createApp",
//		AdminOnly:      false,
//		CanBePerformed: true,
//		Blacklist:      []int{},
//	}, {
//		Service:        service,
//		Name:           "deleteApp",
//		AdminOnly:      false,
//		CanBePerformed: true,
//		Blacklist:      []int{},
//	}, {
//		Service:        service,
//		Name:           "updateApp",
//		AdminOnly:      false,
//		CanBePerformed: true,
//		Blacklist:      []int{},
//	},
//	}
//
//	for _, action := range actions {
//		_, err := conn.Collection("actions").InsertOne(context.Background(), action)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
