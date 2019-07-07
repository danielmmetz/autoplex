FROM golang:alpine

RUN : \
    && apk update \
    && apk add \
        git \
        unrar \
    && :

WORKDIR /autoplex
COPY go.mod go.sum /autoplex/
RUN go mod download
COPY . /autoplex
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o autoplex

ENTRYPOINT ./autoplex
