package main

// var (
// 	cli *client.Client
// )

// func CreateNewContainer(name, image, tag string, natPort int, environment []string) (string, error) {
// 	//set the nat ip to localhost
// 	hostBinding := nat.PortBinding{
// 		HostIP: "0.0.0.0",
// 		//HostPort is the port that the host will listen to, since it's not set
// 		//the docker engine will assign a random open port

// 		// HostPort: "8080",
// 	}

// 	//set the port for the container (internal one)
// 	containerPort, err := nat.NewPort("tcp", strconv.Itoa(natPort))
// 	if err != nil {
// 		return "", fmt.Errorf("unable to bind port: %v", err)
// 	}

// 	//set a slice of possible port bindings,
// 	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}
// 	cont, err := cli.ContainerCreate(
// 		context.Background(),
// 		&container.Config{
// 			Image:  image,
// 			Labels: map[string]string{"vano": "vano"},
// 			Env:    environment,
// 		},
// 		&container.HostConfig{
// 			PortBindings: portBinding,
// 		}, nil, nil, "") //vanoncini.18008-quizz

// 	if err != nil {
// 		return "", err
// 	}

// 	cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
// 	fmt.Printf("container started, id is %s\n", cont.ID)
// 	return cont.ID, nil
// }

// func main() {
// 	fmt.Println(CreateNewContainer("nginx"))
// }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	defer reader.Close()
// 	io.Copy(os.Stdout, reader)

// 	resp, err := cli.ContainerCreate(ctx, &container.Config{
// 		Image: "alpine",
// 		Cmd:   []string{"echo", "hello world"},
// 		Tty:   false,
// 	}, nil, nil, nil, "")
// 	if err != nil {
// 		panic(err)
// 	}

// 	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
// 		panic(err)
// 	}

// 	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
// 	select {
// 	case err := <-errCh:
// 		if err != nil {
// 			panic(err)
// 		}
// 	case <-statusCh:
// 	}

// 	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
// 	if err != nil {
// 		panic(err)
// 	}

// 	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
// }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	imageName := "bfirsh/reticulate-splines"

// 	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer out.Close()
// 	io.Copy(os.Stdout, out)

// 	resp, err := cli.ContainerCreate(ctx, &container.Config{
// 		Image: imageName,
// 	}, nil, nil, nil, "")
// 	if err != nil {
// 		panic(err)
// 	}

// 	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
// 		panic(err)
// 	}

// 	fmt.Println(resp.ID)
// }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, container := range containers {
// 		fmt.Println(container)
// 	}
// }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, container := range containers {
// 		fmt.Print("Stopping container ", container.ID[:10], "... ")
// 		if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
// 			panic(err)
// 		}
// 		fmt.Println("Success")
// 	}
// }

// func main() {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}

// 	options := types.ContainerLogsOptions{ShowStdout: true}
// 	// Replace this ID with a container that really exists
// 	out, err := cli.ContainerLogs(ctx, "e2d52b8e7e", options)
// 	if err != nil {
// 		panic(err)
// 	}

// 	io.Copy(os.Stdout, out)
// }
