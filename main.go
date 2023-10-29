package main

import (
	"log"

	"github.com/Kavantix/go-form/database"
	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

type ViewContext struct {
	View string
	Data interface{}
}

func main() {
	err := database.Connect("db", "postgres", "postgres", "postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	database.Debug()

	r := gin.Default()

	r.GET("/", HandleUsersIndex)
	{
		r := r.Group("/users")
		r.GET("", HandleUsersIndex)
		r.POST("", HandleCreateUser)
		r.GET("/create", HandleUsersCreate)
		r.GET("/:id", HandleUsersView)
		r.POST("/:id", HandleUpdateUser)
	}
	{
		r := r.Group("/assignments")
		r.GET("", HandleAssignmentsIndex)
	}

	r.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
