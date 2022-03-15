package main

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

type db struct {
	Type          string
	Image         string
	Port          string
	Env           []string
	ConnectionUri string
	Comment       string
}

//should implmenet the other dbms
func TestCreateNewDB(t *testing.T) {
	c, _ := NewController()

	dbs := []db{
		{
			Comment: "mysql only root",
			Type:    "mysql",
			Image:   "mysql",
			Port:    "3306",
			Env: []string{
				"MYSQL_ROOT_PASSWORD=root",
			},
			ConnectionUri: "root:root@tcp(127.0.0.1:%s)/",
		},
		{
			Comment: "mysql with database",
			Type:    "mysql",
			Image:   "mysql",
			Port:    "3306",
			Env: []string{
				"MYSQL_ROOT_PASSWORD=root",
				"MYSQL_DATABASE=test",
			},
			ConnectionUri: "root:root@tcp(127.0.0.1:%s)/test",
		},
		{
			Comment: "mysql with user",
			Type:    "mysql",
			Image:   "mysql",
			Port:    "3306",
			Env: []string{
				"MYSQL_ROOT_PASSWORD=root",
				"MYSQL_USER=test",
				"MYSQL_PASSWORD=test",
			},
			ConnectionUri: "test:test@tcp(127.0.0.1:%s)/",
		},
	}

	for _, db := range dbs {
		fmt.Printf("creating %s... ", db.Comment)
		id, err := c.CreateNewDB(db.Type, db.Image, db.Port, db.Env)
		if err != nil {
			t.Errorf("error creating db has been generated: %s", err)
		}
		fmt.Printf("%s\n", id)

		externalPort, err := c.GetContainerExternalPort(id, db.Port)
		if err != nil {
			t.Errorf("error getting external port: %s", err)
		}

		//check the connection
		switch db.Type {
		case "mysql":
			fmt.Printf("checking connection of %s... ", db.Comment)
			_, err := sql.Open("mysql", fmt.Sprintf(db.ConnectionUri, externalPort))
			if err != nil {
				t.Errorf("\nerror connecting to db: %s", err)
			}
			fmt.Println("ok")
		}
	}

	//remove all docker except the ipaas database
	//docker rm $(docker stop $(docker ps -a | grep -v "ipaas_db_1" | awk 'NR>1 {print $1}'))
}
