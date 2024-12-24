package clientid

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

const (
	HeaderKeyClientID = "X-Client-Id"
	maxDomainLength   = 100
)

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

	if clientData.Source != "" {
		return clientData.Source
	}

	// Fallback to source extraction
	clientSource := fallbackClientSource(c)

	return clientSource
}

func fallbackClientSource(c *gin.Context) string {
	if origin := c.GetHeader("Origin"); origin != "" {
		if domain := extractDomain(origin); domain != "" {
			return "null:" + truncateDomain(domain)
		}
	}

	if referer := c.GetHeader("Referer"); referer != "" {
		if domain := extractDomain(referer); domain != "" {
			return "null:" + truncateDomain(domain)
		}
	}

	return ""
}

func extractDomain(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" || urlStr == "null" {
		return ""
	}

	if u, err := url.Parse(urlStr); err == nil {
		return u.Hostname()
	}

	return ""
}

func truncateDomain(domain string) string {
	if len(domain) > maxDomainLength {
		return domain[:maxDomainLength]
	}

	return domain
}
