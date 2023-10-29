package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/templates"
	"github.com/gin-gonic/gin"
)

func HandleUsersIndex(c *gin.Context) {
	page, pageSize, err := paginationParams(c)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}
	users, err := database.GetUsers(page, pageSize)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	template(c, 200, templates.UsersOverview(users))
}

func HandleUsersView(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(400, err)
		return
	}
	user, err := database.GetUser(userId)
	if err == database.ErrNotFound {
		c.Status(404)
		return
	}
	template(c, 200, templates.UsersView(user))
}

func HandleUsersCreate(c *gin.Context) {
	template(c, 200, templates.UsersCreate(nil, map[string]string{}))
}

func HandleUpdateUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(400, err)
		return
	}
	user, err := database.GetUser(userId)
	if err != nil {
		if err == database.ErrNotFound {
			c.Status(404)
			return
		}
		c.AbortWithError(500, err)
		return
	}
	user.Name = c.PostForm("name")
	user.Email = c.PostForm("email")
	user.DateOfBirth, err = time.Parse("2006-01-02", c.PostForm("date_of_birth"))
	if err != nil {
		c.AbortWithError(400, fmt.Errorf("failed to parse date of birth: %w", err))
		return
	}
	err = database.UpdateUser(userId, user.Name, user.Email, user.DateOfBirth)
	if err != nil {
		if err == database.ErrDuplicateEmail {
			c.Request.Context()
			template(c, 422, templates.UsersCreate(user, map[string]string{
				"email": "Email already used",
			}))
			return
		} else {
			c.AbortWithError(500, err)
			return
		}
	}
	c.Header("hx-push-url", "/users")
	HandleUsersIndex(c)
}

func HandleCreateUser(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	dateOfBirth, err := time.Parse("2006-01-02", c.PostForm("date_of_birth"))
	if err != nil {
		c.AbortWithError(400, fmt.Errorf("failed to parse date of birth: %w", err))
		return
	}
	userId, err := database.CreateUser(name, email, dateOfBirth)
	if err != nil {
		if err == database.ErrDuplicateEmail {
			c.Request.Context()
			c.Header("hx-replace-url", "/users/create")
			template(c, 422, templates.UsersCreate(&database.UserRow{
				Name:        name,
				Email:       email,
				DateOfBirth: dateOfBirth,
			}, map[string]string{
				"email": "Email already used",
			}))
			return
		} else {
			c.AbortWithError(500, err)
			return
		}
	}
	fmt.Printf("Created user with id %d\n", userId)
	user, err := database.GetUser(userId)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Header("hx-push-url", fmt.Sprintf("/users/%d", userId))
	template(c, 201, templates.UsersView(user))
}

func HandleAssignmentsIndex(c *gin.Context) {
	page, pageSize, err := paginationParams(c)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}
	assignments, err := database.GetAssignments(page, pageSize)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	template(c, 200, templates.AssignmentOverview(assignments))
}
