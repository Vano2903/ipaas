package utils

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidGithubUrl(t *testing.T) {
	assertions := assert.New(t)
	urls := []struct {
		Url         string
		ShouldError bool
	}{
		{"    github.com/vano2903/testing", false},
		{"github.com/vano2903/testing.git     ", false},
		{"www.github.com/vano2903/testing", false},
		{"www.github.com/vano2903/testing.git    ", false},
		{"https://github.com/vano2903/testing", false},
		{"     https://github.com/vano2903/testing.git", false},
		{"gitlab.com/vano2903/testing", true},
		{"github.com/vano2903/this-repo-doesnt-exist", true},
		{"github/vano2903/this-url-is-not-valid", true},
	}

	for _, url := range urls {
		err := ValidGithubUrl(url.Url)
		if !url.ShouldError {
			if err != nil {
				t.Errorf("error validating %s: %v", url.Url, err)
			}
		} else {
			assertions.Error(err)
		}
	}
}

func TestDownloadGithubRepo(t *testing.T) {
	assertions := assert.New(t)

	urls := []struct {
		Url         string
		Branch      string
		ShouldError bool
	}{
		{"https://github.com/vano2903/testing", "master", false},
		{"github.com/vano2903/testing", "kill-on-command-version", false},
		{"github.com/vano2903/testing", "non-working-version", false},
		{"github.com/vano2903/testing", "runtime-error", false},
		{"github.com/vano2903/testing", "inexisting-branch", true},
	}

	userID := 10000
	var paths []string
	for _, url := range urls {
		path, _, _, err := DownloadGithubRepo(userID, url.Branch, url.Url)
		if !url.ShouldError {
			assertions.NoError(err)
			paths = append(paths, path)
		} else {
			fmt.Println(err)
			assertions.Error(err)
		}
	}

	t.Cleanup(func() {
		//remove the tmp folder
		for _, path := range paths {
			if err := os.RemoveAll(path); err != nil {
				fmt.Printf("error removing path: %s, reason: %v\n", path, err)
			}
		}
	})
}

func TestGetUserAndNameFromRepoUrl(t *testing.T) {
	assertions := assert.New(t)
	urls := []struct {
		Url         string
		Name        string
		Repo        string
		ShouldError bool
	}{
		{"    github.com/vano2903/testing", "vano2903", "testing", false},
		{"github.com/vano2903/testing.git     ", "vano2903", "testing", false},
		{"www.github.com/vano2903/testing", "vano2903", "testing", false},
		{"www.github.com/vano2903/testing.git    ", "vano2903", "testing", false},
		{"https://github.com/vano2903/testing", "vano2903", "testing", false},
		{"     https://github.com/vano2903/testing.git", "vano2903", "testing", false},
		{"gitlab.com/vano2903/testing", "", "", true},
		{"github.com/vano2903/this-repo-doesnt-exist", "", "", true},
		{"github/vano2903/this-url-is-not-valid", "", "", true},
	}

	for _, url := range urls {
		name, repo, err := GetUserAndNameFromRepoUrl(url.Url)
		if !url.ShouldError {
			if err != nil {
				t.Errorf("error validating %s: %v", url.Url, err)
			}
		} else {
			assertions.Error(err)
		}
		assertions.Equal(name, url.Name)
		assertions.Equal(repo, url.Repo)
	}
}

func TestConnectToDB(t *testing.T) {
	testUri := ""
	u := NewUtil(context.Background(), testUri)
	assertions := assert.New(t)
	connection, err := u.ConnectToDB()
	assertions.NoError(err)

	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {
			fmt.Printf("error disconnecting from db: %v\n", err)
		}
	}(connection.Client(), context.Background())
	assertions.NotNil(connection)

	err = connection.Client().Ping(context.Background(), nil)
	assertions.NoError(err)
}

func TestGenerateRandomString(t *testing.T) {
	assertions := assert.New(t)
	lengths := []int{0, 1, 10, 24, 100, 1000, 100000}
	for _, length := range lengths {
		randomString := GenerateRandomString(length)
		assertions.Equal(length, len(randomString))
	}
}
