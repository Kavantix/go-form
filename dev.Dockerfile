FROM golang:1.22.0
RUN go install github.com/mitranim/gow@latest
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download

CMD gow -v -r=false -e=go,mod,html,env,css,js run .
