package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	_ "github.com/go-sql-driver/mysql"
)

var (
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
	MYSQL_IMAGE  = "mysql"
	MYSQL_PORT   = "3306"
)

func CreateNewDB(image, port string, env []string) (string, error) {
	//set the context and client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}

	//pull the image, it wont pull it if it is already there
	//it will update itself since if there is no tag by default it means latest
	out, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer out.Close()
	//!could be used to sent to the client the output of the pull
	// io.Copy(os.Stdout, out)

	//!container configurations

	//container config
	config := &container.Config{
		Image: image,
		Env:   env,
	}

	//host config
	hostBinding := nat.PortBinding{
		HostIP: "0.0.0.0",
		//HostPort is the port that the host will listen to, since it's not set
		//the docker engine will assign a random open port
		// HostPort: "8080",
	}

	//set the port for the container (internal one)
	containerPort, err := nat.NewPort("tcp", port)
	if err != nil {
		return "", err
	}

	//set a slice of possible port bindings
	//since it's a db container we need just one
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	//set the configuration of the host
	//set the port bindings and the restart policy
	//!choose a restart policy
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 5,
		},
	}

	//create the container
	//!set a name to identify the container (<student-name>.<registration_number>-<db-name>)
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	//start the container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

//function to generate a random alphanumerical password without spaces
func generatePassword() string {
	minNum := 4
	minUpperCase := 4
	passwordLength := 16

	rand.Seed(time.Now().Unix())
	var password strings.Builder

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func main() {

	password := generatePassword()
	fmt.Println("password:", password)

	env := []string{
		"MYSQL_ROOT_PASSWORD=" + password,
		"MYSQL_USER=test",
		"MYSQL_PASSWORD=test",
		"MYSQL_DATABASE=test",
	}

	fmt.Println("creating new db")
	id, _ := CreateNewDB(MYSQL_IMAGE, MYSQL_PORT, env)
	fmt.Println(id)

	//get the port that the container is listening to
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	container, err := cli.ContainerInspect(context.Background(), "c61f796566ce")
	if err != nil {
		panic(err)
	}

	fmt.Println("getting port")
	port := container.NetworkSettings.NetworkSettingsBase.Ports["3306/tcp"][0].HostPort

	username := "test"
	pass := "test"
	dbName := "test"
	fmt.Println("username", username)
	fmt.Println("password", pass)
	fmt.Println("database", dbName)
	fmt.Println("port", port)

	//connect to mysql
	fmt.Println("connecting to mysql")
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(localhost:%s)/%s", username, pass, port, dbName))
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		panic(err.Error())
	}

	fmt.Println("YOOOO FUNZIONA TUTTOOOOOOOOOOOOOO")
}
