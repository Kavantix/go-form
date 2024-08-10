package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/Kavantix/go-form/auth"
	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/mails"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting...")
	isProduction := IsProduction(LookupEnv("ENVIRONMENT", "dev") == "production")
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

	gin.SetMode(gin.ReleaseMode)

	log.Println("Configuring routes...")
	r := gin.Default()
	r.UseH2C = true
	r.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	r.SetTrustedProxies([]string{})
	r.Use(gzip.Gzip(gzip.BestSpeed))

	RegisterRoutes(
		r,
		disk,
		isProduction,
		queries,
	)

	RegisterMailhogProxy(
		r,
		MailhogHost(mailhogHost),
		MailhogUser(MustLookupEnv("MAILHOG_USER")),
		MailhogPassword(MustLookupEnv("MAILHOG_PASSWORD")),
	)

	port := LookupEnv("PORT", "80")
	log.Printf("Listening op port %s\n", port)
	log.Fatalln(r.Run(fmt.Sprintf(":%s", port)))
}
