package valueobject

import (
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type RouteCacheMode string

const (
	RouteCacheModePoint         = "token"
	RouteCacheModeRangeByUSD    = "usd"
	RouteCacheModeRangeByAmount = "amount"
)

const (
	RouteCacheKeyTokensDelimiter        = "-"
	RouteCacheKeyDexesDelimiter         = "-"
	RouteCacheKeyExcludedPoolsDelimiter = "-"
)

// RouteCacheKey contains data to build route cache key
type RouteCacheKey struct {
	TokenIn   string
	TokenOut  string
	SaveGas   bool
	CacheMode string
	// AmountIn can be calculated in usd (if token in has price) or amountIn without decimal (if token in has no price)
	AmountIn               string
	Dexes                  []string
	GasInclude             bool
	IsPathGeneratorEnabled bool
	IsHillClimbingEnabled  bool
	ExcludedPools          []string
}

type RouteCacheKeyTTL struct {
	Key *RouteCacheKey
	TTL time.Duration
}

// String receives prefix and returns cache key
func (k *RouteCacheKey) String(prefix string) string {
	return utils.Join(
		prefix,
		strings.Join([]string{k.TokenIn, k.TokenOut}, RouteCacheKeyTokensDelimiter),
		k.SaveGas,
		k.CacheMode,
		k.AmountIn,
		strings.Join(k.Dexes, RouteCacheKeyDexesDelimiter),
		k.GasInclude,
		k.IsPathGeneratorEnabled,
		k.IsHillClimbingEnabled,
		strings.Join(k.ExcludedPools, RouteCacheKeyExcludedPoolsDelimiter),
	)
}

func (k *RouteCacheKey) Hash(prefix string) uint64 {
	return xxhash.Sum64String(k.String(prefix))
}
