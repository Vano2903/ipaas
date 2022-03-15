package main

import "testing"

//should implmenet the other dbms
func TestCreateNewDB(t *testing.T) {
	c, _ := NewController()

	_, err := c.CreateNewDB("mysql", "mysql", "3306", []string{
		"MYSQL_ROOT_PASSWORD=ciao",
	})
	if err != nil {
		t.Errorf("error has been generated: %s", err)
	}
}
