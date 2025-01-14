package valueobject

import (
	"encoding/binary"
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
	TokenIn        string
	TokenOut       string
	OnlySinglePath bool
	CacheMode      string
	// AmountIn can be calculated in usd (if token in has price) or amountIn without decimal (if token in has no price)
	AmountIn                  string
	Dexes                     []string
	GasInclude                bool
	ExcludedPools             []string
	Index                     string
	UseKyberPrivateLimitOrder bool
}

type RouteCacheKeyTTL struct {
	Key *RouteCacheKey
	TTL time.Duration
}

// String receives prefix and returns cache key
func (k *RouteCacheKey) String(prefix string) string {
	args := []interface{}{
		prefix,
		strings.Join([]string{k.TokenIn, k.TokenOut}, RouteCacheKeyTokensDelimiter),
		k.OnlySinglePath,
		k.CacheMode,
		k.AmountIn,
		strings.Join(k.Dexes, RouteCacheKeyDexesDelimiter),
		k.GasInclude,
		strings.Join(k.ExcludedPools, RouteCacheKeyExcludedPoolsDelimiter),
	}
	if k.Index != "" {
		args = append(args, k.Index)
	}
	return utils.Join(args...)
}

// Hash produces a quick statistically unique hash for RouteCacheKey. This hash is NOT cryptographically secure.
func (k *RouteCacheKey) Hash(prefix string) uint64 {
	d := xxhash.New()
	_, _ = d.WriteString(prefix)
	_, _ = d.WriteString(k.TokenIn)
	_, _ = d.WriteString(k.TokenOut)
	if k.OnlySinglePath {
		_, _ = d.Write([]byte{1})
	}
	_, _ = d.WriteString(k.CacheMode)
	_, _ = d.WriteString(k.AmountIn)
	if k.Index != "" {
		_, _ = d.WriteString(k.Index)
	}
	dexHash := uint64(0)
	for _, dex := range k.Dexes {
		dexHash ^= xxhash.Sum64String(dex)
	}
	for _, pool := range k.ExcludedPools {
		dexHash ^= xxhash.Sum64String(pool)
	}
	_, _ = d.Write(binary.LittleEndian.AppendUint64(nil, dexHash))
	if k.GasInclude {
		_, _ = d.Write([]byte{1})
	}

	return d.Sum64()
}
