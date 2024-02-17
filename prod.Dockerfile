FROM --platform=$BUILDPLATFORM golang:1.22.0 as builder

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY docker/main.go ./
RUN GOOS="linux" GOARCH="amd64" CGO_ENABLED=0 go build -v .

FROM --platform=$BUILDPLATFORM builder as build
COPY . .
RUN GOOS="linux" GOARCH="amd64" CGO_ENABLED=0 go build -v -o /go/bin/go-form .

FROM alpine as server
WORKDIR /app
COPY --from=build /go/bin/go-form /app/go-form
CMD /app/go-form
