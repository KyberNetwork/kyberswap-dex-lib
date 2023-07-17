package requestid

import "github.com/gin-gonic/gin"

const HeaderKeyRequestID = "X-Request-Id"

func ExtractRequestID(c *gin.Context) string {
	return c.GetHeader(HeaderKeyRequestID)
}
