package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func New(skipPathSet map[string]struct{}, logCfg logger.Configuration, logBackend logger.LoggerBackend) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, contained := skipPathSet[c.Request.URL.Path]; contained {
			return
		}

		startTime := time.Now()
		requestID := requestid.ExtractRequestID(c)
		clientID := clientid.ExtractClientID(c)

		span := trace.SpanFromContext(c.Request.Context())

		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		reqBody, _ := io.ReadAll(tee)
		c.Request.Body = io.NopCloser(&buf)

		// build child logger with requestId and set to context to be used later
		commonFields := logger.Fields{
			"request.id": requestID,
		}

		var reqLogger logger.Logger
		isDebugRequest := len(c.Request.Header.Get(constant.DebugHeader)) > 0
		if isDebugRequest {
			// create new logger to not affect other requests
			if lg, err := logger.NewLogger(logCfg, logBackend); err == nil {
				reqLogger = lg.WithFields(commonFields)
				reqLogger.SetLogLevel("debug")
			}
		}

		if reqLogger == nil {
			reqLogger = logger.WithFieldsNonContext(commonFields)
		}

		c.Set(string(constant.CtxLoggerKey), reqLogger)

		reqLogger.WithFields(logger.Fields{
			"request.method":     c.Request.Method,
			"request.uri":        c.Request.URL.RequestURI(),
			"request.body":       string(reqBody),
			"request.client_ip":  c.ClientIP(),
			"request.user_agent": c.Request.UserAgent(),
			"client.id":          clientID,
			"span.id":            span.SpanContext().TraceID().String(),
		}).Info("inbound request")

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		resp, _ := io.ReadAll(blw.body)

		reqLogger.WithFields(
			logger.Fields{
				"response.status":      blw.Status(),
				"response.body":        string(resp),
				"response.duration_ms": time.Since(startTime).Milliseconds(),
			}).
			Info("inbound response")
		metrics.IncrRequestCount(c, clientID, blw.Status())
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
