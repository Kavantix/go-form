package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Kavantix/go-form/auth"
	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/mails"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/Kavantix/go-form/templates/components"
	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func tryGetUserIdFromCookie(c *gin.Context, allowExpired bool) (int32, error) {
	token, err := c.Cookie("goform_auth")
	if err != nil {
		return 0, err
	}
	claims, err := auth.ParseJwt(token)
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

func tryGetUserFromCookie(c *gin.Context, queries *database.Queries, allowExpired bool) (database.DisplayableUser, error) {
	userId, err := tryGetUserIdFromCookie(c, allowExpired)
	if err != nil {
		return database.DisplayableUser{}, err
	}
	user, err := queries.GetUser(c.Request.Context(), userId)
	if err != nil {
		return user, err
	}
	return user, nil
}

func HandleRelogin(queries *database.Queries) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, err := tryGetUserFromCookie(c, queries, true)
		if err != nil {
			htmxRedirect(c, "/login")
			return
		}
		token, err := auth.GenerateOTP(6)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("failed to generate relogin token: %w", err))
			return
		}
		_, err = queries.InsertReloginToken(c.Request.Context(), user.Id, token)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("failed to save relogin token: %w", err))
			return
		}
		mails.Relogin(mails.ReloginMailContent{
			User:  user,
			Token: token,
		}).SendTo(c.Request.Context(), user.Email)
		template(c, 200, templates.ReloginForm(user.Email, "", ""))
	}
}

func HandlePutRelogin(isProduction bool, queries *database.Queries) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, err := tryGetUserFromCookie(c, queries, true)
		if err != nil {
			htmxRedirect(c, "/login")
			return
		}
		tokenInvalid := func(c *gin.Context, token string) {
			c.Header("HX-Reswap", "outerHTML")
			template(c, 422, templates.ReloginForm(user.Email, token, fmt.Sprintf("Token `%s` is invalid", token)))
		}
		token := c.PostForm("token")
		if len(token) != 6 {
			tokenInvalid(c, token)
			return
		}
		createdAfter := time.Now().Add(-time.Minute * 5)
		err = queries.ConsumeReloginToken(c.Request.Context(), user.Id, token, createdAfter)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				tokenInvalid(c, token)
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		}
		err = setUserLoggedInCookie(c, int(user.Id), isProduction)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to create token: %w", err))
			return
		}
		c.Status(200)
	}
}

func HandleLogin(queries *database.Queries) func(c *gin.Context) {
	return func(c *gin.Context) {
		email := ""
		user, err := tryGetUserFromCookie(c, queries, true)
		if err == nil {
			email = user.Email
		}
		template(c, 200, templates.Login(email))
	}
}

func HandlePostLogin(queries *database.Queries) func(c *gin.Context) {
	return func(c *gin.Context) {
		email := c.PostForm("email")
		if email == "" {
			c.AbortWithError(400, fmt.Errorf("email param missing"))
			return
		}
		user, err := queries.GetUserByEmail(c.Request.Context(), email)
		if errors.Is(err, database.ErrNotFound) {
			c.Error(fmt.Errorf("email %s not found", email))
			template(c, 200, templates.LoginMessage())
			return
		} else if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to check if user exists: %w", err))
			return
		}
		token, err := auth.CreateJwt(&auth.JwtOptions{
			Audience: "loginlink",
			Subject:  strconv.Itoa(int(user.Id)),
			ValidFor: time.Minute * 5,
		})
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to create token: %w", err))

			return
		}

		link := fmt.Sprintf("http://%s/loginlink?token=%s",
			c.Request.Host,
			url.QueryEscape(token),
		)

		mails.Login(mails.LoginMailContent{
			User: user,
			Link: link,
		}).SendTo(c.Request.Context(), user.Email)

		template(c, 200, templates.LoginMessage())
	}
}

func HandleLoginLink(isProduction bool, queries *database.Queries) func(c *gin.Context) {
	return func(c *gin.Context) {
		tokenString := c.Query("token")
		if tokenString == "" {
			c.Error(fmt.Errorf("No token provided for login"))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}
		claims, err := auth.ParseJwt(tokenString)
		if err != nil {
			// TODO: show error to user
			c.Error(fmt.Errorf("Invalid token: %w", err))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}
		if claims["aud"] != "loginlink" || claims["sub"] == nil {
			c.Error(fmt.Errorf("Invalid token missing claims"))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}
		rawUserId, ok := claims["sub"].(string)
		if !ok {
			c.Error(fmt.Errorf("No user id in claims"))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}
		userId, err := strconv.Atoi(rawUserId)
		if err != nil {
			c.Error(fmt.Errorf("No valid user id in claims: %w", err))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}
		_, err = queries.GetUser(c.Request.Context(), int32(userId))
		if err != nil {
			c.Error(fmt.Errorf("No valid user id in claims: %w", err))
			c.Abort()
			c.Set("Unauthenticated", true)
			return
		}

		err = setUserLoggedInCookie(c, userId, isProduction)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to create token: %w", err))
			return
		}

		if c.GetBool("isHtmx") {
			htmxRedirect(c, "/users")
		} else {
			c.Redirect(302, "/users")
		}
	}
}

func setUserLoggedInCookie(c *gin.Context, userId int, isProduction bool) error {
	authToken, err := auth.CreateJwt(&auth.JwtOptions{
		Subject:  strconv.Itoa(userId),
		Audience: "go-form",
		ValidFor: time.Hour,
	})
	if err != nil {
		return err
	}

	c.SetCookie(
		"goform_auth",
		authToken,
		// a week in seconds
		3600*24*7,
		"",
		"",
		isProduction,
		true,
	)

	return nil
}

func HandleGetUploadUrl(disk interfaces.DirectUploadDisk) func(c *gin.Context) {
	return func(c *gin.Context) {
		var id uuid.UUID
		var location string
		for {
			id = uuid.New()
			location = fmt.Sprintf("tmp/%s", id.String())
			exists, err := disk.Exists(location)
			if err != nil {
				c.AbortWithError(500, fmt.Errorf("Failed to check if location exists: %w", err))
				return
			}
			if !exists {
				break
			}
		}

		url, err := disk.PutUrl(location)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to create put url: %w", err))
			return
		}
		c.JSON(200, gin.H{
			"id":  id.String(),
			"url": url,
		})
	}
}

func HandleUploadFile(disk interfaces.Disk) func(c *gin.Context) {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.AbortWithStatus(406)
			c.Writer.WriteString("only multipart/form-data allowed")
			return
		}
		files := form.File["file"]
		if files == nil || len(files) == 0 {
			c.AbortWithStatus(400)
			c.Writer.WriteString("missing 'file' part")
			return
		}
		extension := ""
		parts := strings.Split(files[0].Filename, ".")
		if len(parts) > 1 {
			extension = parts[len(parts)-1]
		}
		file, err := files[0].Open()
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to open uploaded file: %w", err))
			return
		}
		defer file.Close()
		id := uuid.New().String()
		location := id
		if extension != "" {
			location = fmt.Sprintf("%s.%s", id, extension)
		}
		err = disk.Put(location, file)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to write uploaded file: %w", err))
			return
		}
		url, err := disk.Url(location)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("Failed to write uploaded file: %w", err))
			return
		}
		c.JSON(201, gin.H{
			"id":  id,
			"url": url,
		})
	}
}
func HandleResourceIndexStream[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		sentry.StartSpan(c.Request.Context(), "mark", sentry.WithDescription("Start processing")).Finish()
		startSseStream(c)
		hasNextPage := true
		page := 0
		pageSize := 50
		for hasNextPage && page < 2 {
			rows, err := resource.FetchPage(c.Request.Context(), page, pageSize)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					c.Error(err)
				}
				c.Abort()
				return
			}
			page += 1
			hasNextPage = len(rows) == pageSize
			start := time.Now()
			select {
			case <-c.Request.Context().Done():
				// "Stream cancelled"
				return
			default:
				templateEvent(c, "row",
					templates.TableRows(resource.TableConfig(), rows),
				)
				sentry.StartSpan(c.Request.Context(), "mark", sentry.WithDescription("Sent first event")).Finish()
				diff := time.Now().Sub(start)
				if diff < time.Millisecond*16 {
					span := sentry.StartSpan(
						c.Request.Context(),
						"sleep",
						sentry.WithDescription("limit throughput"),
					)
					time.Sleep(time.Millisecond*16 - diff)
					span.Finish()
				}
			}
		}
		sendSseEvent(c, "end", "")
		sentry.StartSpan(c.Request.Context(), "mark", sentry.WithDescription("Sent end event")).Finish()
	}
}

type renderPartialOption uint8

const (
	RenderPartial renderPartialOption = 1
	RenderFull    renderPartialOption = 0
)

func HandleResourceIndex[T any](resource resources.Resource[T], partialOption renderPartialOption) func(c *gin.Context) {
	return func(c *gin.Context) {
		handleResourceIndex(c, resource, partialOption)
	}
}

func handleResourceIndex[T any](c *gin.Context, resource resources.Resource[T], partialOption renderPartialOption, extraTemplates ...templ.Component) {
	page, pageSize, err := paginationParams(c)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Millisecond*20)
	defer cancel()
	rows, err := resource.FetchPage(
		ctx,
		page, pageSize,
	)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		c.AbortWithError(500, err)
		return
	}
	templatesToRender := []templ.Component{
		templates.ResourceOverview(resource, rows),
	}
	templatesToRender = append(templatesToRender, extraTemplates...)
	template(c, 200, templatesToRender...)
}

func HandleResourceView[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		row, err := resource.FetchRow(c.Request.Context(), int32(id))
		if err == database.ErrNotFound {
			template(c, 404, templates.NotFound(resource.Location(nil)))
			return
		}
		if templates.IsHtmx(c.Request.Context()) {
			template(c, 200, templates.ResourceView(resource, row, nil))
		} else {
			templateInLayout(c, 200, resource.Location(nil), templates.ResourceView(resource, row, nil))
		}
	}
}

func HandleResourceCreate[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		if templates.IsHtmx(c.Request.Context()) {
			template(c, 200, templates.ResourceCreate(resource, nil, map[string]string{}))
		} else {
			templateInLayout(c, 200, resource.Location(nil), templates.ResourceView(resource, nil, map[string]string{}))
		}
	}
}

func HandleValidateResource[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
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
			formFields[fieldName] = c.Query(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		_, err = resource.ParseRow(c.Request.Context(), id, formFields)
		if err != nil {
			if validationErr, isValidationErr := err.(resources.ValidationError); isValidationErr {
				fmt.Printf("Validation failed %s: %s\n", resource.Title(), err)
				validationErrors[validationErr.FieldName] = validationErr.Message
				c.JSON(422, gin.H{
					"validationErrors": validationErrors,
				})
				return
			} else if parsingErr, isParsingErr := err.(resources.ParsingError); isParsingErr {
				validationErrors := map[string]string{}
				fmt.Printf("Parsing failed %s: %s\n", resource.Title(), err)
				validationErrors[parsingErr.FieldName] = parsingErr.Message
				c.JSON(422, gin.H{
					"validationErrors": validationErrors,
				})
				return
			}
		} else if len(validationErrors) > 0 {
			c.JSON(422, gin.H{
				"validationErrors": validationErrors,
			})
			return
		}
		c.JSON(200, gin.H{
			"validationErrors": gin.H{},
		})
	}
}

func HandleCreateResource[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		validationErrors := map[string]string{}
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.PostForm(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		row, err := resource.ParseRow(c.Request.Context(), nil, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors[validationErr.FieldName] = validationErr.Message
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), parsingErr)
				template(c, 400, templates.ResourceCreate(resource, row, nil))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}

		}
		if len(validationErrors) > 0 {
			template(c, 422,
				templates.ResourceCreate(resource, row, validationErrors),
				components.Toast(components.ToastConfig{
					Message: "Not all fields are valid",
					Variant: components.ToastError,
				}),
			)
			return
		}
		id, err := resource.CreateRow(c.Request.Context(), row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors["email"] = "Email already used"
				c.Header("hx-replace-url", fmt.Sprintf("%s/create", resource.Location(nil)))
				template(c, 200, templates.ResourceCreate(resource, row, validationErrors))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		}
		fmt.Printf("Created %s with id %d\n", resource.Title(), id)
		handleResourceIndex(c, resource, RenderPartial, components.Toast(components.ToastConfig{
			Message: fmt.Sprintf("Sucessfully created %s", resource.Title()),
			Variant: components.ToastSuccess,
		}))
	}
}

func HandleUpdateResource[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		validationErrors := map[string]string{}
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.PostForm(fieldName)
			validationError := field.Validator(formFields[fieldName])
			if validationError != "" {
				validationErrors[fieldName] = validationError
			}
		}
		row, err := resource.ParseRow(c.Request.Context(), &id, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				fmt.Printf("Failed to update %s: %s\n", resource.Title(), validationErr)
				validationErrors[validationErr.FieldName] = validationErr.Reason.Error()
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				fmt.Printf("Failed to update %s: %s\n", resource.Title(), parsingErr)
				template(c, 400, templates.ResourceCreate(resource, row, nil))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		} else if len(validationErrors) > 0 {
			template(c, 422,
				templates.ResourceView(resource, row, validationErrors),
				components.Toast(components.ToastConfig{
					Message: "Not all fields are valid",
					Variant: components.ToastError,
				}),
			)
			return
		}

		err = resource.UpdateRow(c.Request.Context(), row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to update %s: %s\n", resource.Title(), err)
				validationErrors["email"] = "Email already used"
				template(c, 200, templates.ResourceView(resource, row, validationErrors))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		}
		c.Header("hx-push-url", resource.Location(nil))
		handleResourceIndex(c, resource, RenderPartial, components.Toast(components.ToastConfig{
			Message: fmt.Sprintf("Sucessfully updated %s", resource.Title()),
			Variant: components.ToastSuccess,
		}))
	}
}
