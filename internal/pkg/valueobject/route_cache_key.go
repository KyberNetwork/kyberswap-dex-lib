package valueobject

import (
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type RouteCacheMode string

const (
	RouteCacheModePoint = "token"
	RouteCacheModeRange = "usd"
)

const (
	RouteCacheKeyTokensDelimiter = "-"
	RouteCacheKeyDexesDelimiter  = "-"
)

// RouteCacheKey contains data to build route cache key
type RouteCacheKey struct {
	TokenIn    string
	TokenOut   string
	SaveGas    bool
	CacheMode  string
	AmountIn   string
	Dexes      []string
	GasInclude bool
}

// String receives prefix and returns cache key
func (k RouteCacheKey) String(prefix string) string {
	return utils.Join(
		prefix,
		strings.Join([]string{k.TokenIn, k.TokenOut}, RouteCacheKeyTokensDelimiter),
		k.SaveGas,
		k.CacheMode,
		k.AmountIn,
		strings.Join(k.Dexes, RouteCacheKeyDexesDelimiter),
		k.GasInclude,
	)
}
