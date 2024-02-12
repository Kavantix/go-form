package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Kavantix/go-form/auth"
	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/disks"
	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/newdatabase"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func RegisterResource[T any](e *gin.RouterGroup, resource resources.Resource[T]) {
	r := e.Group(resource.Location(nil))
	r.GET("", HandleResourceIndex(resource))
	r.GET("/stream", HandleResourceIndexStream(resource))
	r.GET("/:id", HandleResourceView(resource))
	r.GET("/:id/validate", HandleValidateResource(resource))
	r.GET("/create", HandleResourceCreate(resource))
	r.GET("/validate", HandleValidateResource(resource))
	r.POST("", HandleCreateResource(resource))
	r.POST("/:id", HandleUpdateResource(resource))
}

func MustLookupEnv(key string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		log.Fatalf("Env variable '%s' is required", key)
	}
	return value
}

func LookupEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return fallback
	}
	return value
}

func InitSentry() error {
	templates.FrontendSentryDSN = MustLookupEnv("FRONTEND_SENTRY_DSN")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:                MustLookupEnv("SENTRY_DSN"),
		TracesSampleRate:   1.0,
		EnableTracing:      true,
		ProfilesSampleRate: 1.0,
		Environment:        "local",
	})
	if err != nil {
		return fmt.Errorf("sentry.Init: %s", err)
	}
	return nil
}

func main() {
	isProduction := false
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
		log.Fatalf("Failed ot load keys:\n%s\n", err)
	}

	var disk interfaces.Disk
	uploadDisk := LookupEnv("UPLOAD_DISK", "local")
	switch uploadDisk {
	case "local":
		disk = disks.NewLocal("./storage/public", "/storage", disks.LocalDiskModePublic)
	case "do-spaces":
		disk, err = disks.NewDOSpaces(
			MustLookupEnv("DO_SPACES_REGION"),
			MustLookupEnv("DO_SPACES_BUCKET"),
			MustLookupEnv("DO_SPACES_KEY_ID"),
			MustLookupEnv("DO_SPACES_KEY_SECRET"),
		)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to create s3 disk: %w", err))
		}
	case "s3":
		disk, err = disks.NewS3(
			MustLookupEnv("S3_ENDPOINT"),
			MustLookupEnv("S3_REGION"),
			MustLookupEnv("S3_BASE_URL"),
			MustLookupEnv("S3_BUCKET"),
			MustLookupEnv("S3_KEY_ID"),
			MustLookupEnv("S3_KEY_SECRET"),
			false,
		)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to create s3 disk: %w", err))
		}
	default:
		log.Fatalf("UPLOAD_DISK '%s' is not supported, supported: (locale/s3)", uploadDisk)
	}
	queries, err := database.Connect(
		MustLookupEnv("DB_HOST"),
		LookupEnv("DB_PORT", "5432"),
		MustLookupEnv("DB_USERNAME"),
		MustLookupEnv("DB_PASSWORD"),
		MustLookupEnv("DB_DATABASE"),
		MustLookupEnv("DB_SSLMODE"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	// database.Debug()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	r.SetTrustedProxies([]string{})
	r.Use(gzip.Gzip(gzip.BestSpeed))
	r.Static("/storage", "./storage/public/")
	r.Static("/js", "./public/js/")
	r.Use(func(c *gin.Context) {
		isHtmx := c.GetHeader("HX-Request") == "true"
		ctx := context.WithValue(c.Request.Context(), "isHtmx", isHtmx)
		c.Set("isHtmx", isHtmx)
		*c.Request = *c.Request.WithContext(ctx)
		c.Next()
	})
	r.POST("/upload", HandleUploadFile(disk))
	if disk, ok := disk.(interfaces.DirectUploadDisk); ok {
		r.GET("/upload-url", HandleGetUploadUrl(disk))
	}
	r.Use(func(c *gin.Context) {
		c.Next()
		if c.GetBool("Unauthenticated") {
			if c.GetBool("isHtmx") {
				if c.FullPath() == "/logout" {
					htmxRedirect(c, "/login")
					return
				}
				_, err := tryGetUserIdFromCookie(c, true)
				if err != nil {
					htmxRedirect(c, "/login")
					return
				}
				c.Header("HX-Reswap", "innerHTML show:top")
				c.Header("HX-Retarget", "#relogin")
				template(c, 422, templates.SessionExpired())
			} else {
				c.Redirect(302, "/login")
			}
		}
	})
	r.GET("/loginlink", HandleLoginLink(isProduction, queries))
	authenticated := r.Group("", func(c *gin.Context) {
		userId, err := tryGetUserIdFromCookie(c, false)
		if err != nil {
			c.Set("Unauthenticated", true)
			c.Abort()
			return
		}
		hub := sentry.GetHubFromContext(c.Request.Context())
		hub.Scope().SetUser(sentry.User{
			ID: strconv.Itoa(int(userId)),
		})
		var user *newdatabase.DisplayableUser
		c.Set("GetUser", func() (*newdatabase.DisplayableUser, error) {
			if user == nil {
				*user, err = queries.GetUser(c.Request.Context(), userId)
				if err != nil {
					return nil, err
				}
			}
			return user, err
		})
		c.Next()
	})
	getUser := func(c *gin.Context) (*newdatabase.DisplayableUser, error) {
		user, err := c.MustGet("GetUser").(func() (*newdatabase.DisplayableUser, error))()
		if err != nil {
			c.AbortWithError(500, err)
			return nil, err
		}
		return user, nil
	}
	authenticated.GET("/users/me", func(c *gin.Context) {
		user, err := getUser(c)
		if err != nil {
			return
		}
		c.IndentedJSON(200, user)
	})
	r.GET("/logout", func(c *gin.Context) {
		c.SetCookie(
			"goform_auth",
			"",
			-1,
			"",
			"",
			false,
			true,
		)
		c.Set("Unauthenticated", true)
	})
	RegisterResource(authenticated, resources.NewUserResource(queries))
	RegisterResource(authenticated, resources.NewAssignmentResource())
	r.GET("/login", HandleLogin(queries))
	r.POST("/login", HandlePostLogin(queries))
	r.GET("/relogin", HandleRelogin(queries))
	r.PUT("/relogin", HandlePutRelogin(isProduction, queries))

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/users")
	})
	r.NoRoute(func(c *gin.Context) {
		template(c, 404, templates.NotFound("/users"))
	})

	fmt.Println("Listening op port 80")
	r.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
