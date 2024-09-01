package main

import (
	_ "context"
	_ "errors"
	_ "fmt"
	_ "io/fs"
	_ "log"
	_ "log/slog"
	_ "os"
	_ "runtime"
	_ "strconv"
	_ "strings"
	_ "time"

	_ "github.com/aws/aws-sdk-go-v2/aws"
	_ "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	_ "github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/aws/aws-sdk-go-v2/service/s3/types"
	_ "github.com/aws/smithy-go"
	_ "github.com/getsentry/sentry-go"
	_ "github.com/getsentry/sentry-go/http"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/joho/godotenv"
	_ "github.com/labstack/echo/v4"
	_ "github.com/labstack/echo/v4/middleware"
	_ "github.com/matcornic/hermes/v2"
	_ "github.com/wneessen/go-mail"

	_ "github.com/lib/pq"
)

// empty main file to allow for prebuilding dependencies
func main() {

}
