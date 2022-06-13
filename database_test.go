package main

import (
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
	// c, _ := NewContainerController()

	// _, err := c.CreateNewDB(c.dbContainersConfigs["mysql"], []string{
	// 	"MYSQL_ROOT_PASSWORD=ciao",
	// })
	// if err != nil {
	// 	t.Errorf("error has been generated: %s", err)
	// }

	//remove all docker except the ipaas database
	//docker rm $(docker stop $(docker ps -a | grep -v "ipaas_db_1" | awk 'NR>1 {print $1}'))
}
