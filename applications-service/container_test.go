package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	assertions := assert.New(t)

	controller, err := NewContainerController(context.Background())
	if err != nil {
		t.Fatalf("error creating the new controller: %v", err)
	}

	urls := []struct {
		Url                 string
		Branch              string
		Envs                []Env
		ShouldErrorInBuild  bool
		ShouldErrAtDownload bool
	}{
		{"https://github.com/vano2903/testing", "master", []Env{}, false, false},
		{"github.com/vano2903/testing", "master", []Env{{Key: "TestKey", Value: "testValue"}}, false, false},
		{"github.com/vano2903/testing", "non-working-version", []Env{}, true, false},
		{"github.com/vano2903/testing", "runtime-error", []Env{}, false, false},
		{"github.com/vano2903/testing", "inexisting-branch", []Env{}, true, true},
	}
	userID := 10000
	port := 8080 //webserver port

	var images []string
	var paths []string
	for _, url := range urls {
		userID++
		path, name, _, err := DownloadGithubRepo(userID, url.Branch, url.Url)
		if url.ShouldErrAtDownload {
			assertions.Error(err)
			continue
		} else {
			assertions.NoError(err)
		}
		_, imageID, err := controller.CreateImage(userID, port, name, path, "go", url.Envs)
		if url.ShouldErrorInBuild {
			assertions.Error(err)
		} else {
			assertions.NoError(err)
			//check if the image is created
			images = append(images, imageID)
			fmt.Println(imageID)
		}
		paths = append(paths, path)
	}

	t.Cleanup(func() {
		//delete created images
		for _, image := range images {
			controller.cli.ImageRemove(context.Background(), image, types.ImageRemoveOptions{})
		}
		//delete all files from the tmp folder
		for _, path := range paths {
			os.RemoveAll(path)
		}
	})
}
