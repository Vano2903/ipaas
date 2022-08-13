package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	assert := assert.New(t)

	controller, err := NewContainerController(context.Background())
	if err != nil {
		t.Fatalf("error creating the new controller: %v", err)
	}

	urls := []struct {
		Url         string
		Branch      string
		Envs        []Env
		ShouldError bool
	}{
		{"https://github.com/vano2903/testing", "master", []Env{}, false},
		// {"github.com/vano2903/testing", "master", []Env{{Key: "TestKey", Value: "testValue"}}, false},
		{"github.com/vano2903/testing", "non-working-version", []Env{}, true},
		// {"github.com/vano2903/testing", "runtime-error", []Env{}, false},
		// {"github.com/vano2903/testing", "inexisting-branch", []Env{}, true},
	}
	userID := 10000
	port := 8080 //webserver port

	for _, url := range urls {
		userID++
		path, name, _, _ := DownloadGithubRepo(userID, url.Branch, url.Url)
		_, _, err = controller.CreateImage(userID, port, name, path, "go", url.Envs)
		if url.ShouldError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}

	t.Cleanup(func() {
		//delete created images
		//delete downloaded folders
	})
}
