package clientid

import "github.com/gin-gonic/gin"

const HeaderKeyClientID = "X-Client-Id"

func ExtractClientID(c *gin.Context) string {
	return c.GetHeader(HeaderKeyClientID)
}
