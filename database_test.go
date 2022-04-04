package main

import "testing"

//should implmenet the other dbms
func TestCreateNewDB(t *testing.T) {
	c, _ := NewContainerController()

	_, err := c.CreateNewDB(c.dbContainersConfigs["mysql"], []string{
		"MYSQL_ROOT_PASSWORD=ciao",
	})
	if err != nil {
		t.Errorf("error has been generated: %s", err)
	}
}
