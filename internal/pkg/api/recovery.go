package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func RecoveryFunc(c *gin.Context, err any) {
	log.Ctx(c).Error().Interface("error", err).Msg("internal server error")

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
