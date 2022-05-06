package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	_ "github.com/go-sql-driver/mysql"
)

var (
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
)

type Util struct {
}

//get the student from the database given a valid access token (will be retrived from cookies)
func (u Util) GetUserFromCookie(r *http.Request, connection *sql.DB) (Student, error) {
	//search the access token in the cookies
	var acc string
	for _, cookie := range r.Cookies() {
		switch cookie.Name {
		case "accessToken":
			acc = cookie.Value
		}
	}

	//check if it's not empty
	if acc == "" {
		return Student{}, fmt.Errorf("no access token found")
	}

	//get the student from the database, it will automatically check if the access token is valid
	s, err := GetUserFromAccessToken(acc, connection)
	if err != nil {
		return Student{}, err
	}

	return s, nil
}

//check if a url is a valid github repo download url (github.com/name/example.git)
func (u Util) ValidGithubUrl(url string) bool {
	if !strings.HasPrefix(url, "https://github.com/") {
		return false
	}

	if !strings.HasSuffix(url, ".git") {
		return false
	}

	return true
}

//TODO: branch is not used yet, should be implemented
//this function clone the repository from github given the url and save it in the tmp folder
//it returns the name of the path, name and last commit hash and a possible error
func (u Util) DownloadGithubRepo(userID int, branch, url string) (string, string, string, error) {
	//get the name of the repo
	fmt.Print("getting repo name...")
	repoName := strings.Split(strings.Replace(url, ".git", "", -1), "/")[len(strings.Split(strings.Replace(url, ".git", "", -1), "/"))-1]
	fmt.Println("ok, repo name:", repoName)

	//get the repo name
	fmt.Printf("downloading repo in ./tmp/%d-%s...", userID, repoName)
	r, err := git.PlainClone(fmt.Sprintf("./tmp/%d-%s", userID, repoName), false, &git.CloneOptions{
		URL:          url,
		Depth:        1,
		SingleBranch: true,
		// ReferenceName: plumbing.ReferenceName(branch),
		// Progress: os.Stdout,
	})
	if err != nil {
		return "", "", "", err
	}
	fmt.Println("ok")

	logs, err := r.Log(&git.LogOptions{})
	if err != nil {
		return "", "", "", err
	}
	defer logs.Close()

	//get the last commit hash
	fmt.Print("getting last commit hash...")
	commitHash, err := logs.Next()
	if err != nil {
		return "", "", "", err
	}
	fmt.Println("ok")

	//remove the .git folder
	fmt.Print("removing .git...")
	if err := os.RemoveAll(fmt.Sprintf("./tmp/%d-%s/.git", userID, repoName)); err != nil {
		return "", "", "", err
	}

	fmt.Println("ok")
	return fmt.Sprintf("./tmp/%d-%s", userID, repoName), repoName, commitHash.Hash.String(), nil
}

//generate a new pointer to the util struct
//is like a constructor
func NewUtil() (*Util, error) {
	return &Util{}, nil
}

//returns a pointer to a db connection
func connectToDB() (db *sql.DB, err error) {
	return sql.Open("mysql", "root:root@tcp(localhost:3306)/ipaas?parseTime=true&charset=utf8mb4")
}

//function to generate a random alphanumerical string without spaces and with a given length
func generateRandomString(size int) string {
	minNum := 4
	minUpperCase := 4
	passwordLength := size

	rand.Seed(time.Now().UnixNano())
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

//return a random open port on the host
func getFreePort() (int, error) {
	//:0 is the kernel port to get a free port
	//we are asking for a random open port
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	//listen to the port
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	//close the connection
	defer l.Close()
	//get the port on which the kernel gave us
	return l.Addr().(*net.TCPAddr).Port, nil
}

//remove all continers excet the ipaas db:
//docker rm $(docker ps -a | grep -v "ipaas_db_1" | awk 'NR>1 {print $1}')
