package clientid

import (
	"github.com/gin-gonic/gin"
)

var headerXClientID string

// Config defines the config for ClientID middleware
type config struct {
	headerKey HeaderStrKey
	handler   Handler
}

// New initializes the ClientID middleware.
func New(opts ...Option) gin.HandlerFunc {
	cfg := &config{
		headerKey: "X-Client-ID",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	headerXClientID = string(cfg.headerKey)

	return func(c *gin.Context) {
		// Get client id from request
		cid := c.GetHeader(headerXClientID)
		if cid == "" {
			c.Request.Header.Add(headerXClientID, cid)
		}
		if cfg.handler != nil {
			cfg.handler(c, cid)
		}
		// Set the id to ensure that the client id is in the response
		c.Header(headerXClientID, cid)
		c.Next()
	}
}

// Get returns the request identifier
func Get(c *gin.Context) string {
	return c.Writer.Header().Get(headerXClientID)
}
