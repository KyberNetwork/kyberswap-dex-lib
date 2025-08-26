package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
		botScore := requestid.ExtractBotScore(c)
		clientID := clientid.ExtractClientID(c)

		ctx := c.Request.Context()

		// build child logger with requestId and set to context to be used across the request scope
		lg := log.With().Str("req.id", requestID).Logger()
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
				Str("req.method", c.Request.Method).
				Str("req.uri", c.Request.URL.RequestURI()).
				Stringer("req.body", &reqBodyBuffer).
				Str("req.ip", c.ClientIP()).
				Str("req.ua", c.Request.UserAgent()).
				Str("req.ja4", ja4).
				Int("req.bot", botScore).
				Str("req.client", clientID).
				Int("resp.status", status).
				Stringer("resp.body", &respBodyBuffer).
				Dur("resp.ms", time.Since(startTime)).
				Msg("handled api")
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
