package main

import (
	"context"
	"crypto/subtle"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"strconv"

	"github.com/Kavantix/go-form/database"

	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type IsProduction bool
type MailhogHost string
type MailhogUser string
type MailhogPassword string
type GetUserFunc func(c echo.Context) (*database.DisplayableUser, error)

type echoGroup = echo.Group
type AuthenticatedGroup struct {
	*echoGroup
}

//go:embed public/js
var publicJsFs embed.FS

//go:embed public/css
var publicCssFs embed.FS

func RegisterRoutes(
	r *echo.Echo,
	disk interfaces.Disk,
	isProduction IsProduction,
	queries *database.Queries,
) {
	r.Static("/storage", "./storage/public/")
	jsDir, err := fs.Sub(publicJsFs, "public/js")
	if err != nil {
		log.Fatalf("Failed to create public js dir: %s\n", err)
	}
	r.StaticFS("/js", jsDir)
	cssDir, err := fs.Sub(publicCssFs, "public/css")
	if err != nil {
		log.Fatalf("Failed to create public css dir: %s\n", err)
	}
	r.StaticFS("/css", cssDir)
	r.Use(setIsHtmx)
	// r.POST("/upload", HandleUploadFile(disk))
	// if disk, ok := disk.(interfaces.DirectUploadDisk); ok {
	// 	r.GET("/upload-url", HandleGetUploadUrl(disk))
	// }
	r.Use(handleUnauthenticated)
	r.GET("/loginlink", HandleLoginLink(bool(isProduction), queries))
	authenticated, getUser := setupAuthenticatedGroup(r, queries)
	authenticated.GET("/users/me", func(e echo.Context) error {
		user, err := getUser(e)
		if err != nil {
			return err
		}
		return e.JSONPretty(200, user, "  ")
	})
	r.GET("/logout", HandleLogout())
	r.GET("/login", HandleLogin(queries))
	r.POST("/login", HandlePostLogin(queries))
	// r.GET("/relogin", HandleRelogin(queries))
	// r.PUT("/relogin", HandlePutRelogin(bool(isProduction), queries))

	RegisterResource(authenticated, resources.NewUserResource(queries))
	RegisterResource(authenticated, resources.NewAssignmentResource(queries))

	r.GET("/", func(c echo.Context) error {
		return c.Redirect(302, "/users")
	})
	r.RouteNotFound("/*", func(c echo.Context) error {
		return template(c, 404, templates.NotFound("/users"))
	})
	r.HTTPErrorHandler = func(err error, c echo.Context) {
		he, ok := err.(*echo.HTTPError)
		if ok {
			template(c, he.Code, templates.ServerFailure("/users"))
		} else {
			hub := sentry.GetHubFromContext(c.Request().Context())
			hub.CaptureException(fmt.Errorf("request failed: %w", err))
			template(c, 500, templates.ServerFailure("/users"))
		}
	}
}

func RegisterMailhogProxy(
	r *echo.Echo,
	host MailhogHost,
	correctUser MailhogUser,
	correctPassword MailhogPassword,
) {
	mailhogUrl, err := url.Parse(fmt.Sprintf("http://%s:8025", host))
	if err != nil {
		log.Fatalf("Failed to construct mailhog url: %s\n", err)
	}
	r.Group("/mailhog", middleware.BasicAuth(func(username, password string, ctx echo.Context) (bool, error) {
		if subtle.ConstantTimeCompare([]byte(username), []byte(correctUser)) == 1 &&
			subtle.ConstantTimeCompare([]byte(password), []byte(correctPassword)) == 1 {
			return true, nil
		}
		return false, nil
	}),
		middleware.Proxy(middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				URL: mailhogUrl,
			},
		})),
	)

}

func setIsHtmx(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		isHtmx := c.Request().Header.Get("HX-Request") == "true"
		ctx := context.WithValue(c.Request().Context(), "isHtmx", isHtmx)
		c.Set("isHtmx", isHtmx)
		if isHtmx {
			currentUrl, err := url.Parse(c.Request().Header.Get("HX-Current-URL"))
			if err == nil {
				ctx = context.WithValue(ctx, "currentUrl", currentUrl)
				c.Set("currentUrl", currentUrl)
			}
		}
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func isHtmx(c echo.Context) bool {
	isHtmx, ok := c.Get("isHtmx").(bool)
	return ok && isHtmx
}

func handleUnauthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		handlerErr := next(c)
		_, hasUnauthenticated := c.Get("Unauthenticated").(bool)
		if hasUnauthenticated {
			if isHtmx(c) {
				if c.Path() == "/logout" {
					return htmxRedirect(c, "/login")
				}
				_, err := tryGetUserIdFromCookie(c, true)
				if err != nil {
					return htmxRedirect(c, "/login")
				}
				c.Response().Header().Set("HX-Reswap", "innerHTML show:top")
				c.Response().Header().Set("HX-Retarget", "#relogin")
				return template(c, 422, templates.SessionExpired())
			} else {
				return c.Redirect(302, "/login")
			}
		}
		return handlerErr
	}
}

func setupAuthenticatedGroup(r *echo.Echo, queries *database.Queries) (AuthenticatedGroup, GetUserFunc) {
	group := r.Group("", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userId, err := tryGetUserIdFromCookie(c, false)
			if err != nil {
				c.Set("Unauthenticated", true)
				return nil
			}
			hub := sentry.GetHubFromContext(c.Request().Context())
			hub.Scope().SetUser(sentry.User{
				ID: strconv.Itoa(int(userId)),
			})
			var user *database.DisplayableUser
			c.Set("GetUser", func() (*database.DisplayableUser, error) {
				if user == nil {
					*user, err = queries.GetUser(c.Request().Context(), userId)
					if err != nil {
						return nil, err
					}
				}
				return user, err
			})
			return next(c)
		}
	})
	getUser := func(c echo.Context) (*database.DisplayableUser, error) {
		user, err := c.Get("GetUser").(func() (*database.DisplayableUser, error))()
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		return user, nil
	}
	return AuthenticatedGroup{group}, getUser
}

func RegisterResource[T any](e AuthenticatedGroup, resource resources.Resource[T]) {
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
