package http

import (
	"fmt"
	"net/http"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	gintracer "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	requestidpkg "github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

const (
	readyAPIPath = "/api/v1/health/live"
	liveAPIPath  = "/api/v1/health/ready"
)

func GinServer(cfg *HTTPConfig, zapLogger *zap.Logger) (*gin.Engine, *gin.RouterGroup, error) {
	gin.SetMode(cfg.Mode)
	gin.EnableJsonDecoderUseNumber()

	server := gin.New()
	// just enable datadog if $DD_ENABLED is not empty
	liveAPIPath := getFullPathLiveAPI(cfg.Prefix)
	readyAPIPath := getFullPathReadyAPI(cfg.Prefix)
	middlewares := []gin.HandlerFunc{
		sentrygin.New(
			sentrygin.Options{
				Repanic: true,
			},
		),
		gin.Recovery(),
		ginzap.GinzapWithConfig(zapLogger, &ginzap.Config{SkipPaths: []string{liveAPIPath, readyAPIPath}}),
	}
	if env.StringFromEnv(envvar.DDEnabled, "") != "" {
		middlewares = append(middlewares, gintracer.Middleware(env.StringFromEnv(envvar.DDService, ""), gintracer.WithIgnoreRequest(func(c *gin.Context) bool {
			return c.Request.URL.Path == liveAPIPath || c.Request.URL.Path == readyAPIPath
		})))
	}
	middlewares = append(middlewares,
		requestid.New(
			requestid.WithCustomHeaderStrKey(requestidpkg.RequestIDHeaderKey),
			requestid.WithHandler(func(c *gin.Context, requestID string) {
				c.Request = c.Request.WithContext(requestidpkg.SetRequestIDToContext(c.Request.Context(), requestID))
			}),
		),
		RequestLoggerMiddleware(),
	)

	server.Use(middlewares...)
	setCORS(server)
	server.GET("/ping", func(c *gin.Context) { c.AbortWithStatus(http.StatusOK) })
	router := server.Group(cfg.Prefix)

	return server, router, nil
}

func setCORS(engine *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AddAllowMethods(http.MethodOptions)
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders("Authorization")
	corsConfig.AddAllowHeaders("x-request-id")
	corsConfig.AddAllowHeaders("X-Request-Id")
	corsConfig.AddAllowHeaders("Accept-Version")
	engine.Use(cors.New(corsConfig))
}

func getFullPathReadyAPI(httpPrefix string) string {
	return fmt.Sprintf("%s%s", httpPrefix, readyAPIPath)
}

func getFullPathLiveAPI(httpPrefix string) string {
	return fmt.Sprintf("%s%s", httpPrefix, liveAPIPath)
}
