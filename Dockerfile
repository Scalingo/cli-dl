FROM golang:1.24

RUN go install github.com/cespare/reflex@latest

EXPOSE 20205

WORKDIR $GOPATH/src/github.com/Scalingo/cli-dl
