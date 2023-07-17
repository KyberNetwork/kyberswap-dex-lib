package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func New(skipPathSet map[string]struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, contained := skipPathSet[c.Request.URL.Path]; contained {
			return
		}

		startTime := time.Now()
		requestID := requestid.ExtractRequestID(c)
		clientID := clientid.ExtractClientID(c)

		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		reqBody, _ := io.ReadAll(tee)
		c.Request.Body = io.NopCloser(&buf)

		logger.WithFields(logger.Fields{
			"request.method":     c.Request.Method,
			"request.uri":        c.Request.URL.RequestURI(),
			"request.body":       string(reqBody),
			"request.client_ip":  c.ClientIP(),
			"request.user_agent": c.Request.UserAgent(),
			"request.id":         requestID,
			"client.id":          clientID,
		}).Info("inbound request")

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		resp, _ := io.ReadAll(blw.body)

		logger.WithFields(
			logger.Fields{
				"response.status":      blw.Status(),
				"response.body":        string(resp),
				"response.duration_ms": time.Since(startTime).Milliseconds(),
				"request.id":           requestID,
				"client.id":            clientID,
			}).
			Info("inbound response")
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
