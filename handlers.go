package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

func HandleResourceIndex[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		page, pageSize, err := paginationParams(c)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		rows, err := resource.FetchPage(page, pageSize)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		template(c, 200, templates.ResourceOverview(resource, rows))
	}
}

func HandleResourceView[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		row, err := resource.FetchRow(id)
		if err == database.ErrNotFound {
			c.Status(404)
			return
		}
		template(c, 200, templates.ResourceView(resource, row, nil))
	}
}

func HandleResourceCreate[T any](resource resources.Resource[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		template(c, 200, templates.ResourceCreate(resource, nil, map[string]string{}))
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
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.Query(fieldName)
		}
		_, err = resource.ParseRow(id, formFields)
		if err != nil {
			if validationErr, isValidationErr := err.(resources.ValidationError); isValidationErr {
				validationErrors := map[string]string{}
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
		}
		c.JSON(200, gin.H{
			"validationErrors": gin.H{},
		})
	}
}

func HandleCreateResource[T any](resource resources.Resource[T]) func(c *gin.Context) {
	handleIndex := HandleResourceIndex(resource)
	return func(c *gin.Context) {
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.PostForm(fieldName)
		}
		row, err := resource.ParseRow(nil, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors[validationErr.FieldName] = validationErr.Message
				template(c, 422, templates.ResourceCreate(resource, row, validationErrors))
				return
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), parsingErr)
				template(c, 400, templates.ResourceCreate(resource, row, nil))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}

		}
		id, err := resource.CreateRow(row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors["email"] = "Email already used"
				c.Header("hx-replace-url", fmt.Sprintf("%s/create", resource.Location(nil)))
				template(c, 422, templates.ResourceCreate(resource, row, validationErrors))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		}
		fmt.Printf("Created %s with id %d\n", resource.Title(), id)
		handleIndex(c)
	}
}

func HandleUpdateResource[T any](resource resources.Resource[T]) func(c *gin.Context) {
	handleIndex := HandleResourceIndex(resource)
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		formFields := map[string]string{}
		formConfig := resource.FormConfig()
		for _, field := range formConfig.Fields {
			fieldName := field.Name()
			formFields[fieldName] = c.PostForm(fieldName)
		}
		row, err := resource.ParseRow(&id, formFields)
		if err != nil {
			if validationErr, ok := err.(resources.ValidationError); ok {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to update %s: %s\n", resource.Title(), validationErr)
				validationErrors[validationErr.FieldName] = validationErr.Reason.Error()
				template(c, 422, templates.ResourceView(resource, row, validationErrors))
				return
			} else if parsingErr, ok := err.(resources.ParsingError); ok {
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), parsingErr)
				template(c, 400, templates.ResourceCreate(resource, row, nil))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}

		}
		err = resource.UpdateRow(row)
		if err != nil {
			if err == database.ErrDuplicateEmail {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors["email"] = "Email already used"
				template(c, 422, templates.ResourceView(resource, row, validationErrors))
				return
			} else {
				c.AbortWithError(500, err)
				return
			}
		}
		c.Header("hx-push-url", resource.Location(nil))
		handleIndex(c)
	}
}
