package main

import (
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func template(c *gin.Context, code int, t templ.Component) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	t.Render(c.Request.Context(), c.Writer)
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
