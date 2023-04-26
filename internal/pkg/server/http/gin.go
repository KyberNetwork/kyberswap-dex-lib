package http

import (
	"fmt"
	"net/http"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	gintracer "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"

	"github.com/KyberNetwork/router-service/internal/pkg/api"
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
	skipPathSet := getSkipPathSet(cfg.Prefix)

	middlewares := []gin.HandlerFunc{
		sentrygin.New(
			sentrygin.Options{
				Repanic: true,
			},
		),
	}

	if env.StringFromEnv(envvar.DDEnabled, "") != "" {
		middlewares = append(
			middlewares,
			gintracer.Middleware(
				env.StringFromEnv(envvar.DDService, ""),
				gintracer.WithIgnoreRequest(func(c *gin.Context) bool {
					_, contained := skipPathSet[c.Request.URL.Path]
					return contained
				}),
			),
		)
	}

	middlewares = append(middlewares,
		requestid.New(
			requestid.WithCustomHeaderStrKey(requestidpkg.RequestIDHeaderKey),
			requestid.WithHandler(func(c *gin.Context, requestID string) {
				c.Request = c.Request.WithContext(requestidpkg.SetRequestIDToContext(c.Request.Context(), requestID))
			}),
		),
	)

	server.Use(middlewares...)
	server.Use(LoggerMiddleware(skipPathSet))
	server.Use(gin.CustomRecovery(api.RecoveryFunc))

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

func getSkipPathSet(prefix string) map[string]struct{} {
	skipPathsSet := make(map[string]struct{})

	skipPathsSet[fmt.Sprintf("%s%s", prefix, readyAPIPath)] = struct{}{}
	skipPathsSet[fmt.Sprintf("%s%s", prefix, liveAPIPath)] = struct{}{}

	return skipPathsSet
}
