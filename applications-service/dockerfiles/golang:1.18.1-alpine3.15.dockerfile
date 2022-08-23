FROM golang:1.18.1-alpine3.15
RUN apk add git

ENV IPAAS_APP_NAME %s
ENV IPAAS_REPO %s

%s

WORKDIR /go/src/$IPAAS_APP_NAME

COPY . .
RUN go mod download
RUN go build -o $IPAAS_APP_NAME

EXPOSE %d

CMD ./$IPAAS_APP_NAME