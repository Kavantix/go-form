package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

type IsProduction bool
type MailhogHost string
type MailhogUser string
type MailhogPassword string
type GetUserFunc func(c *gin.Context) (*database.DisplayableUser, error)

type AuthenticatedGroup struct {
	*gin.RouterGroup
}

//go:embed public/js
var publicJsFs embed.FS

//go:embed public/css
var publicCssFs embed.FS

func RegisterRoutes(
	r *gin.Engine,
	disk interfaces.Disk,
	isProduction IsProduction,
	queries *database.Queries,
) {
	r.Static("/storage", "./storage/public/")
	jsDir, err := fs.Sub(publicJsFs, "public/js")
	if err != nil {
		log.Fatalf("Failed to create public js dir: %s\n", err)
	}
	r.StaticFS("/js", http.FS(jsDir))
	cssDir, err := fs.Sub(publicCssFs, "public/css")
	if err != nil {
		log.Fatalf("Failed to create public css dir: %s\n", err)
	}
	r.StaticFS("/css", http.FS(cssDir))
	r.Use(setIsHtmx)
	r.POST("/upload", HandleUploadFile(disk))
	if disk, ok := disk.(interfaces.DirectUploadDisk); ok {
		r.GET("/upload-url", HandleGetUploadUrl(disk))
	}
	r.Use(handleUnauthenticated)
	r.GET("/loginlink", HandleLoginLink(bool(isProduction), queries))
	authenticated, getUser := setupAuthenticatedGroup(r, queries)
	authenticated.GET("/users/me", func(c *gin.Context) {
		user, err := getUser(c)
		if err != nil {
			return
		}
		c.IndentedJSON(200, user)
	})
	r.GET("/logout", func(c *gin.Context) {
		c.SetCookie("goform_auth", "", -1, "", "", false, true)
		c.Set("Unauthenticated", true)
	})
	r.GET("/login", HandleLogin(queries))
	r.POST("/login", HandlePostLogin(queries))
	r.GET("/relogin", HandleRelogin(queries))
	r.PUT("/relogin", HandlePutRelogin(bool(isProduction), queries))

	RegisterResource(authenticated, resources.NewUserResource(queries))
	RegisterResource(authenticated, resources.NewAssignmentResource(queries))

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/users")
	})
	r.NoRoute(func(c *gin.Context) {
		template(c, 404, templates.NotFound("/users"))
	})

}

func RegisterMailhogProxy(
	r *gin.Engine,
	host MailhogHost,
	user MailhogUser,
	password MailhogPassword,
) {
	mailhogUrl, err := url.Parse(fmt.Sprintf("http://%s:8025", host))
	if err != nil {
		log.Fatalf("Failed to construct mailhog url: %s\n", err)
	}
	mailhogProxy := httputil.NewSingleHostReverseProxy(mailhogUrl)
	mailhogBasicAuth := gin.BasicAuth(gin.Accounts{
		string(user): string(password),
	})
	r.Any("/mailhog/*path", mailhogBasicAuth, gin.WrapH(mailhogProxy))

}

func setIsHtmx(c *gin.Context) {
	isHtmx := c.GetHeader("HX-Request") == "true"
	ctx := context.WithValue(c.Request.Context(), "isHtmx", isHtmx)
	c.Set("isHtmx", isHtmx)
	*c.Request = *c.Request.WithContext(ctx)
	c.Next()
}

func handleUnauthenticated(c *gin.Context) {
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
}

func setupAuthenticatedGroup(r *gin.Engine, queries *database.Queries) (AuthenticatedGroup, GetUserFunc) {
	group := r.Group("", func(c *gin.Context) {
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
		var user *database.DisplayableUser
		c.Set("GetUser", func() (*database.DisplayableUser, error) {
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
	getUser := func(c *gin.Context) (*database.DisplayableUser, error) {
		user, err := c.MustGet("GetUser").(func() (*database.DisplayableUser, error))()
		if err != nil {
			c.AbortWithError(500, err)
			return nil, err
		}
		return user, nil
	}
	return AuthenticatedGroup{group}, getUser
}

func RegisterResource[T any](e AuthenticatedGroup, resource resources.Resource[T]) {
	r := e.Group(resource.Location(nil))
	r.GET("", HandleResourceIndex(resource, RenderFull))
	r.GET("/stream", HandleResourceIndexStream(resource))
	r.GET("/:id", HandleResourceView(resource))
	r.GET("/:id/validate", HandleValidateResource(resource))
	r.GET("/create", HandleResourceCreate(resource))
	r.GET("/validate", HandleValidateResource(resource))
	r.POST("", HandleCreateResource(resource))
	r.POST("/:id", HandleUpdateResource(resource))
}
