package main

import (
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/resources"
	"github.com/Kavantix/go-form/templates"
	"github.com/gin-gonic/gin"
)

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
			if err, ok := err.(resources.ValidationError); ok {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to create %s: %s\n", resource.Title(), err)
				validationErrors[err.FieldName] = err.Reason.Error()
				template(c, 422, templates.ResourceCreate(resource, row, validationErrors))
				return
			} else {
				template(c, 400, templates.ResourceCreate(resource, row, nil))
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
			if err, ok := err.(resources.ValidationError); ok {
				validationErrors := map[string]string{}
				fmt.Printf("Failed to update %s: %s\n", resource.Title(), err)
				validationErrors[err.FieldName] = err.Reason.Error()
				template(c, 422, templates.ResourceView(resource, row, validationErrors))
				return
			} else {
				template(c, 400, templates.ResourceView(resource, row, nil))
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
