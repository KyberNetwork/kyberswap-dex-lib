package requestid

import (
	"github.com/KyberNetwork/kutils"
	"github.com/gin-gonic/gin"
)

const (
	HeaderKeyRequestID = "X-Request-Id"
	HeaderKeyJa4       = "X-Ja4"

	QueryKeyBotScore = "x_bot_score"
)

func ExtractRequestID(c *gin.Context) string {
	return c.GetHeader(HeaderKeyRequestID)
}

func ExtractJa4(c *gin.Context) string {
	return c.GetHeader(HeaderKeyJa4)
}

func ExtractBotScore(c *gin.Context) int {
	score, err := kutils.Atoi[int](c.Query(QueryKeyBotScore))
	if err != nil {
		return 100
	}
	return score
}
