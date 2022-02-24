FROM golang:1.17.6-alpine3.14

COPY go.mod go.sum /go/src/ipaas/
RUN go mod download

COPY .env /go/src/ipaas/

COPY *.go /go/src/ipaas/
RUN go build -o ipaas

CMD ./ipaas