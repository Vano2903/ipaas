package main

//RESPONSES

// SuccessResponseStatic is the response for a successful request
// and has a string message in the body.
// The error value will always be false when this struct is used
//
// swagger:model SuccessResponseStatic
type SuccessResponseStatic struct {
	// The http statuc code of the response
	// example: 200
	Code int `json:"code"`

	// The content of the response
	// example: "successfully done something"
	Msg string `json:"msg"`

	// The error is a bool, will tell if the response had an error
	// example: false
	Error bool `json:"error"`
}

// SuccessResponseDynamic is the response for a successful request
// and has a json object in the body (can be a single object or an array).
// The error value will always be false when this struct is used
//
// swagger:model SuccessResponseDynamic
type SuccessResponseDynamic struct {
	// The http statuc code of the response
	// example: 200
	Code int `json:"code"`

	// The content of the response
	// example: "successfully done something"
	Msg string `json:"msg"`

	// The error is a bool, will tell if the response had an error
	// example: false
	Error bool `json:"error"`

	// The dynamic field of the response, can be a single json object or an array of json objects
	// example: {"id": 1, "name": "test"}
	Data string `json:"data"`
}

// ErrorResponseStatic is the response for a failed request
// and has a string message in the body.
// The error value will always be true when this struct is used
//
// swagger:model ErrorResponseStatic
type ErrorResponseStatic struct {
	// The http statuc code of the response
	// example: 400
	Code int `json:"code"`

	// The content of the response
	// example: "something went wrong"
	Msg string `json:"msg"`

	// The error is a bool, will tell if the response had an error
	// example: true
	Error bool `json:"error"`
}

// ErrorResponseDynamic is the response for a failed request
// and has a json object in the body (can be a single object or an array).
// The error value will always be true when this struct is used
//
// swagger:model ErrorResponseDynamic
type ErrorResponseDynamic struct {
	// The http statuc code of the response
	// example: 400
	Code int `json:"code"`

	// The content of the response
	// example: "something went wrong"
	Msg string `json:"msg"`

	// The error is a bool, will tell if the response had an error
	// example: true
	Error bool `json:"error"`

	// The dynamic field of the response, can be a single json object or an array of json objects
	// example: {"id": 1, "name": "test"}
	Data string `json:"data"`
}

// REQUESTS

// ApplicationRequest provides the information needed to create a new application
//
// swagger:model ApplicationRequest
type ApplicationRequest struct {
	// The GitHub url of the application (repo needs to be public)
	// example: github.com/vano2903/testing
	// required: true
	GithubRepoUrl string `json:"github-repo"`

	// The branch of the application
	// example: master
	// required: true
	GithubBranch string `json:"github-branch"`

	// The language used by the application
	// example: go
	// required: true
	Language string `json:"language"`

	// The port on which the application will listen
	// example: 8080
	// required: true
	Port string `json:"port"`

	// The description of the application
	// example: "this application is used for ...."
	// required: false
	Description string `json:"description,omitempty"`

	// The environment variables of the application
	// example: [{key: "port", value: "8080"}, {key: "host", value: "localhost"}]
	// required: false
	Envs []Env `json:"envs,omitempty"`
}

type Env struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
}
