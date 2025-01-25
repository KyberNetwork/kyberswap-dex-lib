package requestid

import "github.com/gin-gonic/gin"

const (
	HeaderKeyRequestID = "X-Request-Id"
	HeaderKeyJa4       = "X-Ja4"
)

func ExtractRequestID(c *gin.Context) string {
	return c.GetHeader(HeaderKeyRequestID)
}

func ExtractJa4(c *gin.Context) string {
	return c.GetHeader(HeaderKeyJa4)
}
