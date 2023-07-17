package clientid

import (
	"github.com/gin-gonic/gin"
)

// Option for queue system
type Option func(*config)

type (
	Handler func(c *gin.Context, requestID string)
)

type HeaderStrKey string

// WithCustomHeaderStrKey set custom header key for request id
func WithCustomHeaderStrKey(s HeaderStrKey) Option {
	return func(cfg *config) {
		cfg.headerKey = s
	}
}

// WithHandler set handler function for request id with context
func WithHandler(handler Handler) Option {
	return func(cfg *config) {
		cfg.handler = handler
	}
}
