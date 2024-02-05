package main

import (
	"fmt"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func template(c *gin.Context, code int, t templ.Component) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	t.Render(c.Request.Context(), c.Writer)
}

func startSseStream(c *gin.Context) {
	c.Status(200)
	c.Writer.Header().Set("Cache-Control", "no-store")
	c.Writer.Header().Set("Content-Type", "text/event-stream")
}

func sendSseEvent(c *gin.Context, name string, data string) {
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", name, data)
	c.Writer.Flush()
}

func templateEvent(c *gin.Context, name string, t templ.Component) {
	fmt.Fprintf(c.Writer, "event: %s\ndata: ", name)
	t.Render(c.Request.Context(), c.Writer)
	fmt.Fprint(c.Writer, "\n\n")
	c.Writer.Flush()
}

func htmxRedirect(c *gin.Context, location string) {
	c.Header("HX-Location", location)
	c.Status(204)
}

func paginationParams(c *gin.Context) (page, pageSize int, err error) {
	page, err = strconv.Atoi(c.DefaultQuery("page", "0"))
	if err != nil {
		return
	}
	pageSize, err = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil {
		return
	}
	return
}
