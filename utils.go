package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/tidwall/gjson"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
	DatabaseUri  string
	JwtSecret    []byte
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

// get the student from the database given a valid access token (will be retrived from cookies)
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

// ValidGithubUrl check if an url is a valid and existing GitHub repo url
// !should allow other git remotes (I.E. gitlab)
func (u Util) ValidGithubUrl(url string) error {
	//sanitize the url
	url = strings.TrimSpace(url)
	url = strings.ToLower(url)

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if !strings.HasPrefix(url, "https://github.") &&
		!strings.HasPrefix(url, "http://github.") &&
		!strings.HasPrefix(url, "https://www.github.") &&
		!strings.HasPrefix(url, "http://www.github.") {
		return errors.New("url is not a github url, make sure it starts with, at least, github.com")
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("invalid url, check if the url is correct or if the repo is not private")
	}

	return nil
}

// GetUserAndNameFromRepoUrl get the username of the creator and the repository's name given a GitHub repository url
func (u Util) GetUserAndNameFromRepoUrl(url string) (string, string, error) {
	err := u.ValidGithubUrl(url)
	if err != nil {
		return "", "", err
	}

	url = strings.TrimSuffix(url, ".git")
	split := strings.Split(url, "/")

	return split[len(split)-2], split[len(split)-1], nil
}

// DownloadGithubRepo clones the repository from GitHub given the url and save it in the tmp folder,
// if the download successfully complete the name of the path, name and last commit hash will be returned
func (u Util) DownloadGithubRepo(userID int, branch, url string) (string, string, string, error) {
	//sanitize the url
	url = strings.TrimSpace(url)
	url = strings.ToLower(url)

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	//get the name of the repo
	fmt.Print("getting repo name...")
	_, repoName, err := u.GetUserAndNameFromRepoUrl(url)
	if err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")
	fmt.Println("repo name:", repoName)

	//get the repo name
	tmpPath := fmt.Sprintf("./tmp/%d-%s-%s", userID, repoName, branch)
	err = os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return "", "", "", fmt.Errorf("error creating the tmp folder: %v", err)
	}
	fmt.Printf("downloading repo in %s...", tmpPath)
	r, err := git.PlainClone(fmt.Sprintf("%s", tmpPath), false, &git.CloneOptions{
		URL:           url,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		// Progress: os.Stdout,
	})
	if err != nil {
		fmt.Println("err")
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
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")

	//remove the .git folder
	fmt.Print("removing .git...")
	if err := os.RemoveAll(fmt.Sprintf("%s/.git", tmpPath)); err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")
	return fmt.Sprintf("%s", tmpPath), repoName, commitHash.Hash.String(), nil
}

// GetMetadataFromRepo gets the description, default branch and all the branches of a GitHub repository
func (u Util) GetMetadataFromRepo(url string) (description, defaultBranch string, branches []string, err error) {
	username, repoName, err := u.GetUserAndNameFromRepoUrl(url)
	if err != nil {
		return "", "", nil, err
	}
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s", username, repoName))
	if err != nil {
		return "", "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", nil, fmt.Errorf("error finding the repository: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, err
	}
	jsonBody := string(body)

	defaultBranch = gjson.Get(jsonBody, "default_branch").String()
	description = gjson.Get(jsonBody, "description").String()
	//get the branches
	resp, err = http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", username, repoName))
	if err != nil {
		return "", "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", nil, fmt.Errorf("error finding the repository: %v", resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, err
	}
	jsonBody = string(body)
	branchesRes := gjson.Get(jsonBody, "@this.#.name").Array()
	for _, r := range branchesRes {
		branches = append(branches, r.String())
	}
	return description, defaultBranch, branches, nil
}

// HasLastCommitChanged will check if the last commit of a GitHub url is different from the given to the function
// TODO: should read just the last one not all the commits in the json
func (u Util) HasLastCommitChanged(commit, url, branch string) (bool, error) {
	//get request to the GitHub api
	owner, name, err := u.GetUserAndNameFromRepoUrl(url)
	if err != nil {
		return false, err
	}

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	//if there is an error return the message
	if resp.StatusCode != 200 {
		var Error GithubCommitInternal
		err = json.Unmarshal(body, &Error)
		if err != nil {
			return false, err
		}
		return false, fmt.Errorf("application: %s/%s returned %d as status code, message: %s", owner, name, resp.StatusCode, Error.Message)
	}

	var RepoCommits []GithubCommit
	err = json.Unmarshal(body, &RepoCommits)
	if err != nil {
		return false, err
	}

	//check if repo doesn't have commits
	if len(RepoCommits) == 0 {
		return false, errors.New("no commit found")
	}

	return RepoCommits[0].SHA != commit, nil
}

// generate a new pointer to the util struct
// is like a constructor
func NewUtil(ctx context.Context) (*Util, error) {
	return &Util{ctx: ctx}, nil
}

// returns a connection to the ipaas database
func connectToDB() (*mongo.Database, error) {
	//get context
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	//try to connect
	clientOptions := options.Client().ApplyURI(DatabaseUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client.Database("ipaas"), nil
}

func initDatabase(db *mongo.Database) error {
	collections := []string{
		"users",
		"applications",
		"langs",
		"oauthStates",
		"pollingIDs",
		"refreshTokens",
	}
	existingCollections, err := db.ListCollectionNames(context.Background(), bson.D{{}})
	if err != nil {
		return err
	}

	//create collections
	for _, collection := range collections {
		//check if collection exists
		fmt.Printf("checking if %s exists\n", collection)
		inexistingCollection := true
		for _, existingCollection := range existingCollections {
			if collection == existingCollection {
				inexistingCollection = false
				break
			}
		}

		if inexistingCollection {
			fmt.Println(collection, "doesn't exists, creating..")

			if err := db.CreateCollection(context.Background(), collection); err != nil {
				return err
			}
		}
	}

	langs, err := db.Collection("langs").CountDocuments(context.Background(), bson.M{})
	if err != nil {
		return err
	}
	if langs == 0 {
		//insert langs
		if _, err := db.Collection("langs").InsertMany(context.Background(), []interface{}{
			bson.M{"lang": "go"},
		}); err != nil {
			return err
		}
	}

	return nil
}

// function to generate a random alphanumerical string without spaces and with a given length
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

// return a random open port on the host
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
