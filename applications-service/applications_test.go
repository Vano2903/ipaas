package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestApp struct {
	App       AppPost
	ShouldErr bool
}

func TestCreateNewApplication(t *testing.T) {
	studentID := 18008

	apps := []TestApp{
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "master",
			Language:      "go",
			Port:          "8080",
			Description:   "working application with environment variables",
			Envs:          []Env{{Key: "TestKey", Value: "testValue"}},
		}, false},
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "runtime-error",
			Language:      "go",
			Port:          "8080",
			Description:   "this application compile but autodestroys itself",
			Envs:          []Env{},
		}, false},

		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "master",
			Language:      "python",
			Port:          "8080",
			Description:   "this language is not supported",
			Envs:          []Env{},
		}, true},
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "master",
			Language:      "go",
			Port:          ":8080",
			Description:   "this port is not valid",
			Envs:          []Env{},
		}, true},
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/inexisting-repo",
			GithubBranch:  "master",
			Language:      "go",
			Port:          "8080",
			Description:   "this repo does not exists",
			Envs:          []Env{},
		}, true},
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "inexisting-branch",
			Language:      "go",
			Port:          "8080",
			Description:   "this branch does not exists",
			Envs:          []Env{},
		}, true},
		{AppPost{
			GithubRepoUrl: "github.com/vano2903/testing",
			GithubBranch:  "non-working-version",
			Language:      "go",
			Port:          "8080",
			Description:   "this application does not compile",
			Envs:          []Env{},
		}, true},
	}

	controller, err := NewContainerController(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertions := assert.New(t)

	var containerIDs []string
	for _, app := range apps {
		appResp, err := controller.CreateNewApplication(studentID, app.App)
		if app.ShouldErr {
			assertions.Error(err)
		} else {
			assertions.NoError(err)
		}

		fmt.Println("description:", app.App.Description)
		fmt.Println("error:", err)
		fmt.Println("resp:", appResp)
		fmt.Println("-----------------------")
		fmt.Println()
		containerIDs = append(containerIDs, appResp.ContainerID)
	}

	t.Cleanup(func() {
		//delete all applications of the user
		for _, id := range containerIDs {
			if err := controller.DeleteApplication(studentID, id); err != nil {
				fmt.Println("error deleting container:", err)
			}
		}
	})
}
