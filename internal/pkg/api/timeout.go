package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func TimeoutHandler(c *gin.Context) {
	c.JSON(
		http.StatusRequestTimeout,
		struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    408,
			Message: "request timeout",
		},
	)
}
