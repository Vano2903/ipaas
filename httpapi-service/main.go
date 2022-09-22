// Package main The API of IPaaS.
//
// IPaaS is a Platform as a Service that let users specify a GitHub repository
// and the service will compile and run the code in a container and expose it.
// It is also possible to create a database and link it to the container.
//
// Version: 1.0.0
// BasePath: /api/v1/
// Schemes: http
// Consumes:
// - application/json
// Produces:
// - application/json
//
// swagger:meta
package main

//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest generate spec -o openapi.yaml
//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest validate openapi.yaml

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/vano2903/ipaas/internal/jwt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/http"
	"os"

	"github.com/vano2903/ipaas/internal/messanger"
	"github.com/vano2903/ipaas/internal/utils"
)

/*
!PAGES ENDPOINTS:
*endpoints for public pages:
/static -> static files
/ -> homepage
/login -> login page
/{studentID} -> public student page
/{studentID}/{appID} -> public app page with info of it

*endpoints for student pages:
/user/ -> user area
/user/application/new -> page to create a new application
/user/database/new -> page to create a new database
! not implemented as pages  /user/application/new -> new application page
! not implemented as pages  /user/database/new -> new database page

!API ENDPOINTS:
*public api endpoints:
/api/oauth -> oauth endpoint (generate the oauth url)
/api/tokens/new -> get a new token pair from a refresh token
/api/{studentID}/all -> get all the public applications of a student
/api/{studentID}/{appID} -> get the info of a public application

*user api endpoints:
/api/user/ -> get the info of the user
/api/user/getApps/{type} -> get all the applications of a user (private or public) with the type of application (database, web, all, updatable)

*container api endpoints:
/api/container/delete/{containerID} -> delete a container
/api/container/publish/{containerID} -> publish a container
/api/container/revoke/{containerID} -> revoke a container

*api endpoints for database:
/api/db/new -> create a new database
! not implemented /api/db/export/{containerID}/{dbName} -> export a database

*api endpoints for applications:
/api/app/new -> create a new application
/api/app/update/{containerID} -> update an application if the repo is changed
*/

const (
	service         = "httpapi-service"
	listeningQueue  = service + "-listening"
	respondingQueue = service + "-responses"
)

var (
	AmpqUrl       string
	MongoUri      string
	MaxGoroutines = 2
	u             *utils.Util
	l             *log.Entry
	parser        *jwt.Parser
	mess          *messanger.Messanger
	appMess       *messanger.Messanger
	handler       *Handler
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
)

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

	//if os.Getenv("MAX_GOROUTINES") != "" {
	//	MaxGoroutines, err = strconv.Atoi(os.Getenv("MAX_GOROUTINES"))
	//	if err != nil {
	//		l.WithFields(log.Fields{
	//			"error": err,
	//		}).Fatal("Error converting MAX_GOROUTINES to int")
	//	}
	//	if MaxGoroutines <= 0 {
	//		l.Fatal("MAX_GOROUTINES must be greater than 0")
	//	}
	//}

	//TODO: key should be env var
	//generate private and public keys
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = &privateKey.PublicKey

	appMess = messanger.NewMessanger(AmpqUrl, "applications-service-listening", "applications-service-responses")
	mess = messanger.NewMessanger(AmpqUrl, listeningQueue, respondingQueue)
	parser = jwt.NewParser([]byte(os.Getenv("JWT_SECRET")))

	l.Info("Starting application service")
	handler = NewHandler()
}

func main() {
	mainRouter := mux.NewRouter()

	//THESE ARE FOR THE STATIC FILES, NOT IMPLEMENTED YET
	//mainRouter.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("static/"))))
	//
	//mainRouter.HandleFunc("/", handler.HomePageHandler)
	//mainRouter.HandleFunc("/login", handler.LoginPageHandler)
	//mainRouter.HandleFunc("/{studentID}", handler.PublicStudentPageHandler)
	//// mainRouter.HandleFunc("/{studentID}/{appID}", handler.PublicAppPageHandler)
	//
	////homepage for the logged user
	//mainRouter.HandleFunc("/user/", handler.UserPageHandler).Methods("GET")
	//
	////user's router with access token middleware
	//userAreaRouter := mainRouter.PathPrefix("/user").Subrouter()
	////set middleware on user area router
	//userAreaRouter.Use(handler.TokensMiddleware)
	////page to create a new application
	//userAreaRouter.HandleFunc("/application/new", handler.NewAppPageHandler).Methods("GET")
	////page to create a new database
	//userAreaRouter.HandleFunc("/database/new", handler.NewDatabasePageHandler).Methods("GET")

	//!API ROUTES
	api := mainRouter.PathPrefix("/api").Subrouter()

	//TODO: host and implement the swagger documentation
	api.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 - Endpoint not existing, check documentation if you are lost :) <DOCUMENTATION_URI)"))
	})

	//! PUBLIC HANDLERS
	api.HandleFunc("/oauth", handler.OauthHandler).Methods("GET")
	api.HandleFunc("/oauth/check/{randomID}", handler.CheckOauthState).Methods("GET")
	api.HandleFunc("/tokens/new", handler.NewTokenPairFromRefreshTokenHandler).Methods("GET")
	api.HandleFunc("/{studentID}/all", handler.GetAllApplicationsOfStudentPublic).Methods("GET")
	// api.HandleFunc("/{studentID}/{appID}", handler.GetInfoApplication).Methods("GET")

	//! USER HANDLERS
	userApiRouter := api.PathPrefix("/user").Subrouter()
	userApiRouter.Use(handler.TokensMiddleware)
	//get the user data
	//still kinda don't know what to do with this one, will probably return the homepage
	userApiRouter.HandleFunc("/", handler.LoginHandler).Methods("GET")
	//get all the applications (even the private one) must define the type (database, web, all)
	userApiRouter.HandleFunc("/getApps/{type}", handler.GetAllApplicationsOfStudentPrivate).Methods("GET")
	////update an application
	//userApiRouter.HandleFunc("/application/update/{containerID}", handler.UpdateApplicationHandler).Methods("GET")

	//! CONTAINER HANDLERS
	containerApiRouter := api.PathPrefix("/container").Subrouter()
	containerApiRouter.Use(handler.TokensMiddleware)
	//delete a container
	containerApiRouter.HandleFunc("/delete/{containerID}", handler.DeleteApplicationHandler).Methods("DELETE")
	//publish a container
	containerApiRouter.HandleFunc("/publish/{containerID}", handler.PublishApplicationHandler).Methods("GET")
	//revoke a container
	containerApiRouter.HandleFunc("/revoke/{containerID}", handler.RevokeApplicationHandler).Methods("GET")

	//! DBaaS HANDLERS
	//DBaaS router (subrouter of user area router so it has access token middleware)
	dbApiRouter := api.PathPrefix("/db").Subrouter()
	dbApiRouter.Use(handler.TokensMiddleware)
	//let the user create a new database
	dbApiRouter.HandleFunc("/new", handler.NewDBHandler).Methods("POST")
	//export a database
	//dbApiRouter.HandleFunc("/export/{containerID}/{dbName}")

	//! APPLICATIONS HANDLERS
	//application router, it's the main part of the application
	appApiRouter := api.PathPrefix("/app").Subrouter()
	appApiRouter.Use(handler.TokensMiddleware)
	appApiRouter.HandleFunc("/new", handler.NewApplicationHandler).Methods("POST")
	appApiRouter.HandleFunc("/update/{containerID}", handler.UpdateApplicationHandler).Methods("POST")

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "POST"})

	log.Println("starting the server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(mainRouter)))
}
