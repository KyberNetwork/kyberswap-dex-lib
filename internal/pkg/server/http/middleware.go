package http

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		body, _ := ioutil.ReadAll(tee)
		c.Request.Body = ioutil.NopCloser(&buf)
		logger.WithFields(logger.Fields{
			"body":   string(body),
			"header": c.Request.Header,
			"query":  c.Request.URL.Query(),
			"path":   c.Request.URL.Path,
		}).Infof("request info for requestId %s", requestid.ExtractRequestID(c))
		c.Next()
	}
}
