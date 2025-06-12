package valueobject

import (
	"encoding/binary"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/samber/lo"

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
	ForcePoolsForToken        map[string][]string
	Index                     string
	UseKyberPrivateLimitOrder bool
	IsScaleHelperClient       bool
	PoolIds                   []string
}

type RouteCacheKeyTTL struct {
	Key *RouteCacheKey
	TTL time.Duration
}

func (k *RouteCacheKey) String() string {
	args := []any{
		strings.Join([]string{k.TokenIn, k.TokenOut}, RouteCacheKeyTokensDelimiter),
		k.OnlySinglePath,
		k.CacheMode,
		k.AmountIn,
		strings.Join(k.Dexes, RouteCacheKeyDexesDelimiter),
		k.GasInclude,
		strings.Join(k.ExcludedPools, RouteCacheKeyExcludedPoolsDelimiter),
		strings.Join(lo.MapToSlice(k.ForcePoolsForToken, func(token string, pools []string) string {
			return token + ":" + strings.Join(pools, "-")
		}), ","),
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
	var unorderedHash uint64
	for _, dex := range k.Dexes {
		unorderedHash ^= xxhash.Sum64String("d" + dex)
	}
	for _, pool := range k.ExcludedPools {
		unorderedHash ^= xxhash.Sum64String("e" + pool)
	}
	for token, pools := range k.ForcePoolsForToken {
		unorderedHash ^= xxhash.Sum64String("f" + token)
		for _, pool := range pools {
			unorderedHash ^= xxhash.Sum64String("o" + pool)
		}
	}
	for _, pool := range k.PoolIds {
		unorderedHash ^= xxhash.Sum64String("p" + pool)
	}
	_, _ = d.Write(binary.LittleEndian.AppendUint64(nil, unorderedHash))
	if k.GasInclude {
		_, _ = d.Write([]byte{1})
	}

	return d.Sum64()
}
