package responser

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func Errorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	Error(w, code, fmt.Sprintf(format, args...))
}

func Error(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true}`, code, message)
}

func ErrorParse(w http.ResponseWriter, code int, message string, values interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, e := json.Marshal(values)
	if e != nil {
		Errorf(w, http.StatusInternalServerError, "Error marshalling json: %s", e)
		return
	}
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func ErrorJson(w http.ResponseWriter, code int, message string, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": true, "data":%s}`, code, message, json)
}

func Successf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	Success(w, code, fmt.Sprintf(format, args...))
}

func Success(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false}`, code, message)
}

func SuccessParse(w http.ResponseWriter, code int, message string, values interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json, e := json.Marshal(values)
	if e != nil {
		Errorf(w, http.StatusInternalServerError, "Error marshalling json: %s", e)
		return
	}
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}

func SuccessJson(w http.ResponseWriter, code int, message string, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "msg":"%s", "error": false, "data":%s}`, code, message, json)
}
