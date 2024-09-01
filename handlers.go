package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Kavantix/go-form/auth"
	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/mails"
	"github.com/Kavantix/go-form/pkg/logger"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/Kavantix/go-form/templates/components"
	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func tryGetUserIdFromCookie(e echo.Context, allowExpired bool) (int32, error) {
	token, err := e.Cookie("goform_auth")
	if err != nil {
		return 0, err
	}
	claims, err := auth.ParseJwt(token.Value)
	if err != nil && (!allowExpired || !errors.Is(err, auth.ErrTokenExpired)) {
		return 0, err
	}
	if claims["aud"] != "go-form" {
		return 0, fmt.Errorf("invalid audience")
	}
	userId, err := strconv.Atoi(claims["sub"].(string))
	if err != nil {
		return 0, err
	}
	return int32(userId), nil
}

func tryGetUserFromCookie(e echo.Context, queries *database.Queries, allowExpired bool) (database.DisplayableUser, error) {
	userId, err := tryGetUserIdFromCookie(e, allowExpired)
	if err != nil {
		return database.DisplayableUser{}, err
	}
	user, err := queries.GetUser(e.Request().Context(), userId)
	if err != nil {
		return user, err
	}
	return user, nil
}

// func HandleRelogin(queries *database.Queries) func(c *gin.Context) {
// 	return func(c *gin.Context) {
// 		user, err := tryGetUserFromCookie(c, queries, true)
// 		if err != nil {
// 			htmxRedirect(c, "/login")
// 			return
// 		}
// 		token, err := auth.GenerateOTP(6)
// 		if err != nil {
// 			c.AbortWithError(500, fmt.Errorf("failed to generate relogin token: %w", err))
// 			return
// 		}
// 		_, err = queries.InsertReloginToken(c.Request.Context(), user.Id, token)
// 		if err != nil {
// 			c.AbortWithError(500, fmt.Errorf("failed to save relogin token: %w", err))
// 			return
// 		}
// 		mails.Relogin(mails.ReloginMailContent{
// 			User:  user,
// 			Token: token,
// 		}).SendTo(c.Request.Context(), user.Email)
// 		template(c, 200, templates.ReloginForm(user.Email, "", ""))
// 	}
// }

// func HandlePutRelogin(isProduction bool, queries *database.Queries) func(c *gin.Context) {
// 	return func(c *gin.Context) {
// 		user, err := tryGetUserFromCookie(c, queries, true)
// 		if err != nil {
// 			htmxRedirect(c, "/login")
// 			return
// 		}
// 		tokenInvalid := func(c *gin.Context, token string) {
// 			c.Header("HX-Reswap", "outerHTML")
// 			template(c, 422, templates.ReloginForm(user.Email, token, fmt.Sprintf("Token `%s` is invalid", token)))
// 		}
// 		token := c.PostForm("token")
// 		if len(token) != 6 {
// 			tokenInvalid(c, token)
// 			return
// 		}
// 		createdAfter := time.Now().Add(-time.Minute * 5)
// 		err = queries.ConsumeReloginToken(c.Request.Context(), user.Id, token, createdAfter)
// 		if err != nil {
// 			if errors.Is(err, database.ErrNotFound) {
// 				tokenInvalid(c, token)
// 				return
// 			} else {
// 				c.AbortWithError(500, err)
// 				return
// 			}
// 		}
// 		err = setUserLoggedInCookie(c, int(user.Id), isProduction)
// 		if err != nil {
// 			c.AbortWithError(500, fmt.Errorf("Failed to create token: %w", err))
// 			return
// 		}
// 		c.Status(200)
// 	}
// }

func HandleLogin(queries *database.Queries) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := ""
		user, err := tryGetUserFromCookie(c, queries, true)
		if err == nil {
			email = user.Email
		}
		return template(c, 200, templates.Login(email))
	}
}

func HandlePostLogin(queries *database.Queries) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		if email == "" {
			return c.String(400, "email param missing")
		}
		user, err := queries.GetUserByEmail(c.Request().Context(), email)
		if errors.Is(err, database.ErrNotFound) {
			logger.EchoInfo(c, "email not found", slog.String("email", email))
			return template(c, 200, templates.LoginMessage())
		} else if err != nil {
			return fmt.Errorf("Failed to check if user exists: %w", err)
		}
		token, err := auth.CreateJwt(&auth.JwtOptions{
			Audience: "loginlink",
			Subject:  strconv.Itoa(int(user.Id)),
			ValidFor: time.Minute * 5,
		})
		if err != nil {
			return fmt.Errorf("Failed to create token: %w", err)
		}

		link := fmt.Sprintf("http://%s/loginlink?token=%s",
			c.Request().Host,
			url.QueryEscape(token),
		)

		mails.Login(mails.LoginMailContent{
			User: user,
			Link: link,
		}).SendTo(c.Request().Context(), user.Email)

		return template(c, 200, templates.LoginMessage())
	}
}

func HandleLoginLink(isProduction bool, queries *database.Queries) func(c echo.Context) error {
	return func(e echo.Context) error {
		tokenString := e.QueryParam("token")
		if tokenString == "" {
			logger.EchoWarn(e, "No token provided for login")
			e.Set("Unauthenticated", true)
			return nil
		}
		claims, err := auth.ParseJwt(tokenString)
		if err != nil {
			// TODO: show error to user
			logger.EchoError(e, "Invalid token: ", err)
			e.Set("Unauthenticated", true)
			return nil
		}
		if claims["aud"] != "loginlink" || claims["sub"] == nil {
			logger.EchoWarn(e, "Invalid token missing claims")
			e.Set("Unauthenticated", true)
			return nil
		}
		rawUserId, ok := claims["sub"].(string)
		if !ok {
			logger.EchoWarn(e, "No user id in claims")
			e.Set("Unauthenticated", true)
			return nil
		}
		userId, err := strconv.Atoi(rawUserId)
		if err != nil {
			logger.EchoError(e, "No valid user id in claims:", err)
			e.Set("Unauthenticated", true)
			return nil
		}
		_, err = queries.GetUser(e.Request().Context(), int32(userId))
		if err != nil {
			logger.EchoError(e, "No valid user id in claims:", err)
			e.Set("Unauthenticated", true)
			return nil
		}

		err = setUserLoggedInCookie(e, userId, isProduction)
		if err != nil {
			e.Error(fmt.Errorf("Failed to create token: %w", err))
			return nil
		}

		if templates.IsHtmx(e.Request().Context()) {
			return htmxRedirect(e, "/users")
		} else {
			return e.Redirect(302, "/users")
		}
	}
}

func HandleLogout() echo.HandlerFunc {
	return func(c echo.Context) error {
		clearUserLoggedInCookie(c)
		c.Set("Unauthenticated", true)
		return nil
	}
}

func clearUserLoggedInCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{

		Name:  "goform_auth",
		Value: "",
		// a week in seconds
		MaxAge:   -1,
		Secure:   false,
		HttpOnly: true,
	},
	)
}

func setUserLoggedInCookie(c echo.Context, userId int, isProduction bool) error {
	authToken, err := auth.CreateJwt(&auth.JwtOptions{
		Subject:  strconv.Itoa(userId),
		Audience: "go-form",
		ValidFor: time.Hour,
	})
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{

		Name:  "goform_auth",
		Value: authToken,
		// a week in seconds
		MaxAge:   3600 * 24 * 7,
		Secure:   isProduction,
		HttpOnly: true,
	},
	)

	return nil
}

func HandleGetUploadUrl(disk interfaces.DirectUploadDisk) echo.HandlerFunc {
	return func(c echo.Context) error {
		var id uuid.UUID
		var location string
		for {
			id = uuid.New()
			location = fmt.Sprintf("tmp/%s", id.String())
			exists, err := disk.Exists(location)
			if err != nil {
				return fmt.Errorf("Failed to check if location exists: %w", err)
			}
			if !exists {
				break
			}
		}

		url, err := disk.PutUrl(location)
		if err != nil {
			return fmt.Errorf("Failed to create put url: %w", err)
		}
		return c.JSON(200, map[string]any{
			"id":  id.String(),
			"url": url,
		})
	}
}

func HandleUploadFile(disk interfaces.Disk) echo.HandlerFunc {
	return func(c echo.Context) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.String(406, "only multipart/form-data allowed")
		}
		files := form.File["file"]
		if files == nil || len(files) == 0 {
			return c.String(400, "missing 'file' part")
		}
		extension := ""
		parts := strings.Split(files[0].Filename, ".")
		if len(parts) > 1 {
			extension = parts[len(parts)-1]
		}
		file, err := files[0].Open()
		if err != nil {
			return fmt.Errorf("Failed to open uploaded file: %w", err)
		}
		defer file.Close()
		id := uuid.New().String()
		location := id
		if extension != "" {
			location = fmt.Sprintf("%s.%s", id, extension)
		}
		err = disk.Put(location, file)
		if err != nil {
			return fmt.Errorf("Failed to write uploaded file: %w", err)
		}
		url, err := disk.Url(location)
		if err != nil {
			return fmt.Errorf("Failed to write uploaded file: %w", err)
		}
		return c.JSON(201, map[string]any{
			"id":  id,
			"url": url,
		})
	}
}
func HandleResourceIndexStream[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !strings.HasPrefix(c.Request().Header.Get("Accept"), "text/event-stream") {
			c.Error(fmt.Errorf("stream requested with wront accept header: %s", c.Request().Header.Get("Accept")))
			return nil
		}
		sentry.StartSpan(c.Request().Context(), "mark", sentry.WithDescription("Start processing")).Finish()
		startSseStream(c)
		hasNextPage := true
		page := 0
		pageSize := 50
		for hasNextPage && page < 2 {
			rows, err := resource.FetchPage(c.Request().Context(), page, pageSize)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return fmt.Errorf("failed to fetch page: %w", err)
			}
			page += 1
			hasNextPage = len(rows) == pageSize
			start := time.Now()
			select {
			case <-c.Request().Context().Done():
				// "Stream cancelled"
				return nil
			default:
				templateEvent(c, "row",
					templates.TableRows(resource.TableConfig(), rows),
				)
				sentry.StartSpan(c.Request().Context(), "mark", sentry.WithDescription("Sent first event")).Finish()
				diff := time.Now().Sub(start)
				if diff < time.Millisecond*16 {
					span := sentry.StartSpan(
						c.Request().Context(),
						"sleep",
						sentry.WithDescription("limit throughput"),
					)
					time.Sleep(time.Millisecond*16 - diff)
					span.Finish()
				}
			}
		}
		sendSseEvent(c, "end", "")
		sentry.StartSpan(c.Request().Context(), "mark", sentry.WithDescription("Sent end event")).Finish()
		return nil
	}
}

func HandleResourceIndex[T any](resource resources.Resource[T]) func(c echo.Context) error {
	return func(c echo.Context) error {
		return handleResourceIndex(c, resource)
	}
}

func handleResourceIndex[T any](c echo.Context, resource resources.Resource[T], extraTemplates ...templ.Component) error {
	page, pageSize, err := paginationParams(c)
	if err != nil {
		logger.EchoInfo(c, "Failed to parse paginationParams", slog.String("error", err.Error()))
		c.String(400, "invalid pagination")
		return nil
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), time.Millisecond*20)
	defer cancel()
	rows, err := resource.FetchPage(
		ctx,
		page, pageSize,
	)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("fetching rows failed: %w", err)
	}
	templatesToRender := []templ.Component{
		templates.ResourceOverview(resource, rows),
	}
	templatesToRender = append(templatesToRender, extraTemplates...)
	return template(c, 200, templatesToRender...)
}

func HandleResourceView[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.String(400, "invalid id")
		}
		row, err := resource.FetchRow(c.Request().Context(), int32(id))
		if err == database.ErrNotFound {
			return template(c, 404, templates.NotFound(resource.Location(nil)))
		}
		if isHtmx(c) {
			return template(c, 200, templates.ResourceView(resource, row, nil))
		} else {
			return templateInLayout(c, 200, resource.Location(nil), templates.ResourceView(resource, row, nil))
		}
	}
}

func HandleResourceCreate[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isHtmx(c) {
			return template(c, 200, templates.ResourceCreate(resource, nil, map[string]string{}))
		} else {
			return templateInLayout(c, 200, resource.Location(nil), templates.ResourceView(resource, nil, map[string]string{}))
		}
	}
}

func HandleValidateResource[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		idParam, err := strconv.Atoi(c.Param("id"))
		var id *int
		if err == nil {
			id = &idParam
		}
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		validationErrors := map[string]string{}
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.QueryParam(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		_, err = resource.ParseRow(c.Request().Context(), id, formFields)
		if err != nil {
			if validationErr, isValidationErr := err.(resources.ValidationError); isValidationErr {
				logger.EchoInfo(c, "Validation failed\n", slog.String("resource", resource.Title()), slog.String("reason", err.Error()))
				validationErrors[validationErr.FieldName] = validationErr.Message
				return c.JSON(422, map[string]any{
					"validationErrors": validationErrors,
				})

			} else if parsingErr, isParsingErr := err.(resources.ParsingError); isParsingErr {
				validationErrors := map[string]string{}
				logger.EchoInfo(c, "Parsing failed\n", slog.String("resource", resource.Title()), slog.String("reason", parsingErr.Error()))
				validationErrors[parsingErr.FieldName] = parsingErr.Message
				return c.JSON(422, map[string]any{
					"validationErrors": validationErrors,
				})
			}
		} else if len(validationErrors) > 0 {
			return c.JSON(422, map[string]any{
				"validationErrors": validationErrors,
			})
		}
		return c.JSON(200, map[string]any{
			"validationErrors": map[string]any{},
		})
	}
}

func HandleCreateResource[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		validationErrors := map[string]string{}
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.FormValue(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		row, err := resource.ParseRow(c.Request().Context(), nil, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				logger.EchoInfo(c, "Validation failed\n", slog.String("resource", resource.Title()), slog.String("reason", err.Error()))
				validationErrors[validationErr.FieldName] = validationErr.Message
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				logger.EchoInfo(c, "Parsing failed\n", slog.String("resource", resource.Title()), slog.String("reason", parsingErr.Error()))
				return template(c, 400, templates.ResourceCreate(resource, row, nil))
			} else {
				return fmt.Errorf("failed to parse row %w", err)
			}

		}
		if len(validationErrors) > 0 {
			return template(c, 422,
				templates.ResourceCreate(resource, row, validationErrors),
				components.Toast(components.ToastConfig{
					Message: "Not all fields are valid",
					Variant: components.ToastError,
				}),
			)
		}
		id, err := resource.CreateRow(c.Request().Context(), row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				logger.EchoInfo(c, "Duplicate email", slog.String("resource", resource.Title()), slog.String("reason", err.Error()))
				validationErrors["email"] = "Email already used"
				c.Response().Header().Set("hx-replace-url", fmt.Sprintf("%s/create", resource.Location(nil)))
				return template(c, 200, templates.ResourceCreate(resource, row, validationErrors))
			} else {
				return fmt.Errorf("failed to create row", err)
			}
		}
		logger.EchoInfo(c, "Created %s with id %d\n", slog.String("resource", resource.Title()), slog.Int("id", int(id)))
		return handleResourceIndex(c, resource, components.Toast(components.ToastConfig{
			Message: fmt.Sprintf("Sucessfully created %s", resource.Title()),
			Variant: components.ToastSuccess,
		}))
	}
}

func HandleUpdateResource[T any](resource resources.Resource[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.String(400, "invalid id")
		}
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		validationErrors := map[string]string{}
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.FormValue(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		row, err := resource.ParseRow(c.Request().Context(), &id, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				logger.EchoInfo(c, "Validation failed\n", slog.String("resource", resource.Title()), slog.String("reason", err.Error()))
				validationErrors[validationErr.FieldName] = validationErr.Reason.Error()
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				logger.EchoInfo(c, "Parsing failed\n", slog.String("resource", resource.Title()), slog.String("reason", parsingErr.Error()))
				return template(c, 400, templates.ResourceCreate(resource, row, nil))
			} else {
				return fmt.Errorf("failed to parse row: %s", err)
			}
		} else if len(validationErrors) > 0 {
			err := triggerToast(c, ToastConfig{
				Message: "Not all fields are valid",
				Variant: ToastError,
			})
			if err != nil {
				logger.EchoError(c, "failed to show toast", err)
			}
			return template(c, 200, templates.ResourceView(resource, row, validationErrors))
		}

		err = resource.UpdateRow(c.Request().Context(), row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				logger.EchoInfo(c, "Failed to update", slog.String("resource", resource.Title()), slog.String("reason", err.Error()))
				validationErrors["email"] = "Email already used"
				return template(c, 200, templates.ResourceView(resource, row, validationErrors))
			} else {
				return fmt.Errorf("failed to update row: %w", err)
			}
		}
		c.Response().Header().Set("hx-push-url", resource.Location(nil))
		return handleResourceIndex(c, resource, components.Toast(components.ToastConfig{
			Message: fmt.Sprintf("Sucessfully updated %s", resource.Title()),
			Variant: components.ToastSuccess,
		}))
	}
}
