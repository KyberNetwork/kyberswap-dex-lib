package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func GetConfig(
	getConfig func() map[string]any,
) func(ginCtx *gin.Context) {
	return func(ginCtx *gin.Context) {
		ginCtx.JSON(http.StatusOK, maskConfig(getConfig()))
	}
}

var sensitiveKeysRegex = regexp.MustCompile(`authorization|(auth|api|priv(ate)?|sec(ret)?)_?key`)

func maskConfig(config map[string]any) map[string]any {
	return lo.MapValues(config, func(value any, key string) any {
		switch subValue := value.(type) {
		case map[string]any:
			return maskConfig(subValue)
		case string:
			if sensitiveKeysRegex.MatchString(key) {
				return "*****"
			}
		}
		return value
	})
}
