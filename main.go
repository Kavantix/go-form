package main

import (
	"log"

	"github.com/Kavantix/go-form/database"
	"github.com/Kavantix/go-form/resources"
	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

func RegisterResource[T any](e *gin.Engine, resource resources.Resource[T]) {
	r := e.Group(resource.Location(nil))
	r.GET("", HandleResourceIndex(resource))
	r.GET("/:id", HandleResourceView(resource))
	r.GET("/create", HandleResourceCreate(resource))
	r.POST("", HandleCreateResource(resource))
	r.POST("/:id", HandleUpdateResource(resource))
}

func main() {
	err := database.Connect("db", "postgres", "postgres", "postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	database.Debug()

	r := gin.Default()
	RegisterResource(r, resources.UserResource{})
	RegisterResource(r, resources.AssignmentResource{})

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/users")
	})

	r.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
