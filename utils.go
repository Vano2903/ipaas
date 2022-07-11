package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
	DATABASE_URI string
	JWT_SECRET   []byte
)

type Util struct {
	ctx context.Context
}

type GithubCommit struct {
	SHA    string               `json:"sha"`
	Commit GithubCommitInternal `json:"commit"`
}

type GithubCommitInternal struct {
	Message string `json:"message"`
}

//get the student from the database given a valid access token (will be retrived from cookies)
func (u Util) GetUserFromCookie(r *http.Request, connection *mongo.Database) (Student, error) {
	//search the access token in the cookies
	var acc string
	for _, cookie := range r.Cookies() {
		switch cookie.Name {
		case "ipaas-access-token":
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
		// ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
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

//given a github repository url it will return the name of the repo and the owner
func (u Util) GetUserAndNameFromRepoUrl(url string) (string, string) {
	url = url[19 : len(url)-4]
	fmt.Println("new URL:", url)
	split := strings.Split(url, "/")
	return split[0], split[1]
}

//TODO: should read just the last one not all the commits in the json
//given the github information of a repo it will tell if the commit has changed in the remote repo
func (u Util) HasLastCommitChanged(commit, url, branch string) (bool, error) {
	//get request to the github api
	owner, name := u.GetUserAndNameFromRepoUrl(url)
	fmt.Printf("https://api.github.com/repos/%s/%s/commits?sha=%s\n", owner, name, branch)
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?sha=%s", owner, name, branch), nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	//read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	fmt.Println("body 1:", string(body))

	//if there is an error return the message
	if resp.StatusCode != 200 {
		var Error GithubCommitInternal
		err = json.Unmarshal(body, &Error)
		if err != nil {
			return false, err
		}
		return false, fmt.Errorf("application: %s/%s returned a status code: %d, message: %s", owner, name, resp.StatusCode, Error.Message)
	}

	var RepoCommits []GithubCommit
	err = json.Unmarshal(body, &RepoCommits)
	if err != nil {
		return false, err
	}

	//check if repo dosen't have commits
	if len(RepoCommits) == 0 {
		return false, errors.New("no commit found")
	}

	return RepoCommits[0].SHA != commit, nil
}

//generate a new pointer to the util struct
//is like a constructor
func NewUtil(ctx context.Context) (*Util, error) {
	return &Util{ctx: ctx}, nil
}

//returns a connection to the ipaas database
func connectToDB() (*mongo.Database, error) {
	//get context
	ctx, _ := context.WithTimeout(context.TODO(), 10*time.Second)

	//try to connect
	clientOptions := options.Client().ApplyURI(DATABASE_URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client.Database("ipaas"), nil
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
	addr, err := net.ResolveTCPAddr("tcp", os.Getenv("IP")+":0")
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
