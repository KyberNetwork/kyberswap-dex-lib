package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AbortWith500(c *gin.Context, message string) {
	RespondWith(c, http.StatusInternalServerError, message, nil)
}

func RespondWith(c *gin.Context, code int, msg string, data interface{}) {
	c.AbortWithStatusJSON(code, data)
}

func AbortWith400(c *gin.Context, message string) {
	RespondWith(c, http.StatusBadRequest, message, nil)
}
