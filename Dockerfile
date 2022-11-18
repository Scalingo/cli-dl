FROM golang:1.17

RUN go install github.com/cespare/reflex@latest

EXPOSE 20205

WORKDIR $GOPATH/src/github.com/Scalingo/cli-dl
