package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
)

func New(skipPathSet map[string]struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, contained := skipPathSet[c.Request.URL.Path]; contained {
			return
		}

		startTime := time.Now()
		requestID := requestid.ExtractRequestID(c)
		ja4 := requestid.ExtractJa4(c)
		clientID := clientid.ExtractClientID(c)

		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)

		// build child logger with requestId and set to context to be used across the request scope
		lg := log.With().Str("request.id", requestID).Logger()
		if c.Request.Header.Get(constant.DebugHeader) != "" {
			lg = lg.Level(zerolog.DebugLevel)
		}
		c.Request = c.Request.WithContext(lg.WithContext(ctx))

		var reqBodyBuffer, respBodyBuffer bytes.Buffer
		c.Request.Body = &readCloser{Reader: io.TeeReader(c.Request.Body, &reqBodyBuffer), Closer: c.Request.Body}
		c.Writer = &bodyLogWriter{logWriter: &respBodyBuffer, ResponseWriter: c.Writer}

		defer func() {
			status := c.Writer.Status()
			lg.Info().
				Str("request.method", c.Request.Method).
				Str("request.uri", c.Request.URL.RequestURI()).
				Stringer("request.body", &reqBodyBuffer).
				Str("request.client_ip", c.ClientIP()).
				Str("request.user_agent", c.Request.UserAgent()).
				Str("request.ja4", ja4).
				Str("client.id", clientID).
				Stringer("span.id", span.SpanContext().TraceID()).
				Int("response.status", status).
				Stringer("response.body", &respBodyBuffer).
				Dur("response.duration_ms", time.Since(startTime)).
				Msg("handled request")
			metrics.CountRequest(c, clientID, ja4, status)
		}()

		c.Next()
	}
}

type readCloser struct {
	io.Reader
	io.Closer
}

type bodyLogWriter struct {
	gin.ResponseWriter
	logWriter *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.logWriter.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.logWriter.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
