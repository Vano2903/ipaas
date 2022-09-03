package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

var (
	//password elements
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
)

// type Util struct {
// 	ctx context.Context
// }

type GithubCommit struct {
	SHA    string               `json:"sha"`
	Commit GithubCommitInternal `json:"commit"`
}

type GithubCommitInternal struct {
	Message string `json:"message"`
}

// ValidGithubUrl check if an url is a valid and existing GitHub repo url
// !should allow other git remotes (I.E. gitlab)
func ValidGithubUrl(url string) (bool, error) {

	//trim the url
	url = strings.TrimSpace(url)

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if !strings.HasPrefix(url, "https://github.") &&
		!strings.HasPrefix(url, "http://github.") &&
		!strings.HasPrefix(url, "https://www.github.") &&
		!strings.HasPrefix(url, "http://www.github.") {
		return false, errors.New("url is not a github url")
	}

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, errors.New("invalid url, check if the url is correct or if the repo is not private")
	}

	return true, nil
}

// DownloadGithubRepo clones the repository from GitHub given the url and save it in the tmp folder,
// if the download successfully complete the name of the path, name and last commit hash will be returned
func DownloadGithubRepo(userID int, branch, url string) (string, string, string, error) {
	url = strings.TrimSpace(url)

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	//get the name of the repo
	fmt.Print("getting repo name...")
	_, repoName, err := GetUserAndNameFromRepoUrl(url)
	if err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")
	fmt.Println("repo name:", repoName)

	//get the repo name
	tmpPath := fmt.Sprintf("./tmp/%d-%s-%s", userID, repoName, branch)
	os.Mkdir(tmpPath, os.ModePerm)
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

// GetUserAndNameFromRepoUrl get the username of the creator and the repository's name given a GitHub repository url
func GetUserAndNameFromRepoUrl(url string) (string, string, error) {
	url = strings.TrimSpace(url)
	//fmt.Println("getting user and name from url:", url)
	valid, err := ValidGithubUrl(url)
	if err != nil {
		return "", "", err
	}
	if !valid {
		return "", "", errors.New("invalid url")
	}

	url = strings.TrimSuffix(url, ".git")
	split := strings.Split(url, "/")

	return split[len(split)-2], split[len(split)-1], nil
}

// HasLastCommitChanged will check if the last commit of a GitHub url is different from the given to the function
// TODO: should read just the last one not all the commits in the json
func HasLastCommitChanged(commit, url, branch string) (bool, error) {
	//get request to the GitHub api
	owner, name, err := GetUserAndNameFromRepoUrl(url)
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

	fmt.Println("body 1:", string(body))

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

// //generate a new pointer to the util struct
// //is like a constructor
// func NewUtil(ctx context.Context) (*Util, error) {
// 	return &Util{ctx: ctx}, nil
// }

// ConnectToDB returns a connection to the ipaas database
func ConnectToDB() (*mongo.Database, error) {
	//get context
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	//try to connect
	clientOptions := options.Client().ApplyURI(MongoUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client.Database("ipaas"), nil
}

// GenerateRandomString will generate a random alphanumerical string of the given length
func GenerateRandomString(size int) string {
	rand.Seed(time.Now().UnixNano())
	var password strings.Builder
	for i := 0; i < size; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
