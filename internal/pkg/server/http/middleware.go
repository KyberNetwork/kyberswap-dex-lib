package http

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		body, _ := io.ReadAll(tee)
		c.Request.Body = io.NopCloser(&buf)
		logger.WithFields(logger.Fields{
			"body":   string(body),
			"header": c.Request.Header,
			"query":  c.Request.URL.Query(),
			"path":   c.Request.URL.Path,
		}).Infof("request info for requestId %s", requestid.ExtractRequestID(c))
		c.Next()
	}
}
