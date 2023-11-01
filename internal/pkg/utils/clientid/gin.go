package clientid

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

const HeaderKeyClientID = "X-Client-Id"

func ExtractClientID(c *gin.Context) string {
	// Extract ClientID from header. Support for API v2.
	clientFromHeader := c.GetHeader(HeaderKeyClientID)
	if clientFromHeader != "" {
		return clientFromHeader
	}

	// Extract ClientID from clientData query. Support for Legacy API.
	clientDataStr := c.Query("clientData")
	var clientData struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(clientDataStr), &clientData); err != nil {
		return ""
	}
	return clientData.Source
}
