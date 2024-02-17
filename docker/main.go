package main

import (
	_ "context"
	_ "errors"
	_ "fmt"
	_ "io/fs"
	_ "log"
	_ "os"
	_ "strconv"
	_ "strings"

	_ "github.com/getsentry/sentry-go"
	_ "github.com/getsentry/sentry-go/gin"
	_ "github.com/gin-contrib/gzip"
	_ "github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// empty main file to allow for prebuilding dependencies
func main() {

}
