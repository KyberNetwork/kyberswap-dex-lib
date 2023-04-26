package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

func RecoveryFunc(c *gin.Context, err any) {
	logger.
		WithFields(logger.Fields{"err": err}).
		Error("internal server error")

	c.JSON(
		http.StatusInternalServerError,
		struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    500,
			Message: "internal server error",
		},
	)
}
