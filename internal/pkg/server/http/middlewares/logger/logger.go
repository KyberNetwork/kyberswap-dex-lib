package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
)

func New(skipPathSet map[string]struct{}, logCfg klog.Configuration, logBackend klog.LoggerBackend) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, contained := skipPathSet[c.Request.URL.Path]; contained {
			return
		}

		startTime := time.Now()
		requestID := requestid.ExtractRequestID(c)
		clientID := clientid.ExtractClientID(c)

		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		reqBody, _ := io.ReadAll(tee)
		c.Request.Body = io.NopCloser(&buf)

		// build child logger with requestId and set to context to be used later
		commonFields := klog.Fields{
			"request.id": requestID,
		}

		var reqLogger klog.Logger
		isDebugRequest := len(c.Request.Header.Get(constant.DebugHeader)) > 0
		if isDebugRequest {
			// create new logger to not affect other requests
			if lg, err := klog.NewLogger(logCfg, logBackend); err == nil {
				reqLogger = lg.WithFields(commonFields)
				_ = reqLogger.SetLogLevel("debug")
			}
		}

		if reqLogger == nil {
			reqLogger = klog.WithFields(ctx, commonFields)
		}

		c.Set(string(constant.CtxLoggerKey), reqLogger)

		reqLogger.WithFields(klog.Fields{
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
			klog.Fields{
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
