FROM golang:1.18.0-alpine3.15

ENV IPAAS_APP_NAME %s
ENV IPAAS_REPO %s

%s


WORKDIR /go/src/$IPAAS_APP_NAME

COPY . .
RUN go mod download
RUN go build -o $IPAAS_APP_NAME

EXPOSE %d

CMD ./$IPAAS_APP_NAME