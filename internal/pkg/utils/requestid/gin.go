package requestid

import "github.com/gin-gonic/gin"

const RequestIDHeaderKey = "X-Request-Id"

func ExtractRequestID(c *gin.Context) string {
	return c.GetHeader(RequestIDHeaderKey)
}
