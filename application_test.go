package main

import (
	"context"
	"testing"
)

func TestCreateNewApplicationFromRepo(t *testing.T) {
	c, _ := NewContainerController()
	u, _ := NewUtil(context.Background())
	var tmpPath, imageName, imageID string //, containerID
	var err error

	t.Run("Downloading repo to tmp path", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing.git"
		creatorID := 18008
		branch := "non-working-version"
		t.Log("repo:", repo)
		t.Log("creatorID:", creatorID)
		t.Log("branch:", branch)
		if err := u.ValidGithubUrl(repo); err != nil {
			t.Fatalf("%s is not a valid github url", repo)
		}
		var name, lastCommit string
		tmpPath, name, lastCommit, err = u.DownloadGithubRepo(creatorID, branch, repo)
		if err != nil {
			t.Fatalf("error has been generated: %s", err)
		}
		t.Logf("tmpPath: %s", tmpPath)
		t.Logf("name: %s", name)
		t.Logf("lastCommit: %s", lastCommit)
	})

	// //!check if the image fails
	t.Run("Creating the image from repo", func(t *testing.T) {
		creatorID := 18008
		port := 8080
		name := "test"
		language := "go"
		branch := "master"

		imageName, imageID, err = c.CreateImage(creatorID, port, name, branch, tmpPath, language, nil)
		if err != nil {
			t.Fatalf("error has been generated: %s", err)
		}
		t.Logf("imageName: %s", imageName)
		t.Logf("imageID: %s", imageID)
	})

	// t.Run("Creating a new aplication", func(t *testing.T) {
	// 	creatorID := 18008
	// 	port := "8080"
	// 	name := "test"
	// 	language := "go"
	// 	containerID, err = c.CreateNewApplicationFromRepo(creatorID, port, name, language, imageName)
	// 	if err != nil {
	// 		t.Fatalf("error has been generated creating a container: %s", err)
	// 	}
	// 	t.Log("containerID:", containerID)
	// })

	// t.Cleanup(func() {
	// 	err = c.DeleteContainer(containerID)
	// 	if err != nil {
	// 		t.Fatalf("error has been generated deleting a container: %s", err)
	// 	}
	// 	t.Log("deleted container successfully")
	// 	err = c.RemoveImage(imageID)
	// 	if err != nil {
	// 		t.Fatalf("error has been generated deleting an image: %s", err)
	// 	}
	// 	t.Log("deleted image successfully")
	// 	err = os.RemoveAll(tmpPath)
	// 	if err != nil {
	// 		t.Fatalf("error has been generated deleting a tmp path: %s", err)
	// 	}
	// 	t.Log("deleted tmp path successfully")
	// })
}
