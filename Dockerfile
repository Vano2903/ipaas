FROM golang:1.17.8-alpine3.14

WORKDIR /go/src/ipaas/
RUN mkdir tmp
COPY dockerfiles/ /go/src/ipaas/dockerfiles/
COPY responser/ /go/src/ipaas/responser/

COPY go.mod go.sum /go/src/ipaas/
RUN go mod download 

COPY .env /go/src/ipaas/

COPY *.go /go/src/ipaas/
RUN go build -o ipaas

CMD ./ipaas