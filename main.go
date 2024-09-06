package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"

	"github.com/Kavantix/go-form/auth"
	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/mails"
	"github.com/Kavantix/go-form/pkg/env"
	"github.com/Kavantix/go-form/pkg/logger"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()
	isProduction := IsProduction(LookupEnv("ENVIRONMENT", "dev") == "production")

	logger.InitGoogleCloudLogger()

	log.Println("Starting...")
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf("Error loading .env file:\n%s\n", err)
	}

	err = InitSentry()
	if err != nil {
		log.Fatalf("Cannot initialize sentry:\n%s\n", err)
	}

	err = auth.LoadKeys(MustLookupEnv("PRIVATE_KEY"), MustLookupEnv("PUBLIC_KEY"))
	if err != nil {
		log.Fatalf("Failed to load keys:\n%s\n", err)
	}

	mailhogHost := MustLookupEnv("MAILHOG_HOST")
	err = mails.Init(mailhogHost)

	disk := ResolveDisk(LookupEnv, MustLookupEnv)
	queries, err := database.Connect(
		MustLookupEnv("DB_HOST"),
		LookupEnv("DB_PORT", "5432"),
		MustLookupEnv("DB_USERNAME"),
		MustLookupEnv("DB_PASSWORD"),
		MustLookupEnv("DB_DATABASE"),
		MustLookupEnv("DB_SSLMODE"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %s\n", err)
	}
	defer database.Close()
	// database.Debug(database.DebugOptions{
	// 	IncludeValues: false,
	// })

	log.Println("Configuring routes...")
	r := echo.New()
	logger.SetupEchoGoogleCloudLogger(r, env.Lookup("PROJECT_ID", "eighth-gamma-414620"))
	r.Use(echo.WrapMiddleware(sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle))
	r.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		// Add extra middleware for sentry since default implementation does not add the status to the transaction
		return func(c echo.Context) error {
			transaction := sentry.TransactionFromContext(c.Request().Context())
			var err error
			if transaction != nil {
				defer func() {
					transaction.Status = sentry.HTTPtoSpanStatus(c.Response().Status)
					if err != nil {
						//  this block should not be executed in case of HandleError=true as the global error handler will decide
						//  the status code. In that case status code could be different from what err contains.
						var httpErr *echo.HTTPError
						if errors.As(err, &httpErr) {
							transaction.Status = sentry.HTTPtoSpanStatus(httpErr.Code)
						}
					}

				}()
			}
			err = next(c)
			return err
		}
	})

	RegisterMailhogProxy(
		r,
		MailhogHost(mailhogHost),
		MailhogUser(MustLookupEnv("MAILHOG_USER")),
		MailhogPassword(MustLookupEnv("MAILHOG_PASSWORD")),
	)

	RegisterRoutes(
		r,
		disk,
		isProduction,
		queries,
	)

	host := env.Lookup("HOST", "0.0.0.0")
	port := LookupEnv("PORT", "80")
	addr := fmt.Sprintf("%s:%s", host, port)
	logger.Info(ctx, fmt.Sprintf("Listening on %s", addr), slog.String("addr", addr))
	r.HideBanner = true
	r.HidePort = true
	if err := r.Start(addr); err != nil && err != http.ErrServerClosed {
		fmt.Printf("server failed: %s\n", err)
	}

}
