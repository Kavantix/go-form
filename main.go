package main

import (
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
	"github.com/Kavantix/go-form/resources"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func RegisterResource[T any](e *gin.RouterGroup, resource resources.Resource[T]) {
	r := e.Group(resource.Location(nil))
	r.GET("", HandleResourceIndex(resource))
	r.GET("/:id", HandleResourceView(resource))
	r.GET("/:id/validate", HandleValidateResource(resource))
	r.GET("/create", HandleResourceCreate(resource))
	r.GET("/create/validate", HandleValidateResource(resource))
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

func main() {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf("Error loading .env file:\n%s\n", err)
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
	err = database.Connect("db", "postgres", "postgres", "postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	database.Debug()

	r := gin.Default()
	r.Static("/storage", "./storage/public/")
	r.Static("/js", "./public/js/")
	r.POST("/upload", HandleUploadFile(disk))
	if disk, ok := disk.(interfaces.DirectUploadDisk); ok {
		r.GET("/upload-url", HandleGetUploadUrl(disk))
	}
	r.GET("/loginlink", HandleLoginLink(false))
	r.Use(func(c *gin.Context) {
		fmt.Println("--------------------- test 1 ")
		c.Next()
		fmt.Println("--------------------- test 2")
		if c.GetBool("Unauthenticated") {
			c.Redirect(302, "/login")
		}
	})
	authenticated := r.Group("", func(c *gin.Context) {
		authToken, err := c.Cookie("goform_auth")
		if err != nil {
			c.Set("Unauthenticated", true)
			c.Abort()
			return
		}
		claims, err := auth.ParseJwt(authToken)
		if err != nil {
			c.Set("Unauthenticated", true)
			c.Abort()
			return
		}
		if claims["aud"] != "go-form" {
			c.Error(fmt.Errorf("Invalid audience"))
			c.Set("Unauthenticated", true)
			c.Abort()
			return
		}
		userId, err := strconv.Atoi(claims["sub"].(string))
		if err != nil {
			c.Error(fmt.Errorf("Invalid user id"))
			c.Set("Unauthenticated", true)
			c.Abort()
			return
		}
		var user *database.UserRow
		c.Set("GetUser", func() (*database.UserRow, error) {
			if user == nil {
				user, err = database.GetUser(userId)
				if err != nil {
					return nil, err
				}
			}
			return user, err
		})
		c.Next()
	})
	getUser := func(c *gin.Context) (*database.UserRow, error) {
		user, err := c.MustGet("GetUser").(func() (*database.UserRow, error))()
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
	authenticated.GET("/logout", func(c *gin.Context) {
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
	RegisterResource(authenticated, resources.UserResource{})
	RegisterResource(authenticated, resources.AssignmentResource{})
	r.GET("/login", HandleLogin())

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/users")
	})

	r.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
