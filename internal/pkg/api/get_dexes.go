package api

import "github.com/gin-gonic/gin"

type GetDexesResponse struct {
	Dexes []string `json:"dexes"`
}

// GetDexes [GET /dexes] Get enable dexes
func GetDexes(dexes []string) func(c *gin.Context) {
	return func(ctx *gin.Context) {
		RespondSuccess(ctx, GetDexesResponse{Dexes: dexes})
	}
}
