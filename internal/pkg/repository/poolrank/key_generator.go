package poolrank

import (
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type KeyGenerator struct {
	prefix string
}

func NewKeyGenerator(prefix string) *KeyGenerator {
	return &KeyGenerator{
		prefix: prefix,
	}
}

func (g *KeyGenerator) GlobalSortedSetKey(sortBy string) string {
	return utils.Join(g.prefix, sortBy)
}

// directPairKey generates key of direct pairs
func (g *KeyGenerator) DirectPairKey(sortBy, token0, token1 string) string {
	return utils.Join(g.prefix, sortBy, g.joinTokens(token0, token1))
}

func (g *KeyGenerator) DirectPairKeyWithoutSort(sortBy, token0, token1 string) string {
	return utils.Join(g.prefix, sortBy, strings.Join([]string{token0, token1}, "-"))
}

// whitelistToWhitelistPairKey generates key of pairs between whitelisted tokens
func (g *KeyGenerator) WhitelistToWhitelistPairKey(sortBy string) string {
	return utils.Join(g.prefix, sortBy, KeyWhitelist)
}

// whitelistToTokenPairKey generates key of pairs between a whitelisted token and a non-whitelisted token
func (g *KeyGenerator) WhitelistToTokenPairKey(sortBy, token string) string {
	return utils.Join(g.prefix, sortBy, KeyWhitelist, token)
}

// whitelistToTokenPairKey generates key of pairs between a whitelisted token and a non-whitelisted token
func (g *KeyGenerator) TokenToWhitelistPairKey(sortBy, token string) string {
	return utils.Join(g.prefix, sortBy, token, KeyWhitelist)
}

func (g *KeyGenerator) joinTokens(token0, token1 string) string {
	// Benchmark: https://freshman.tech/snippets/go/string-concatenation/
	if token0 > token1 {
		return strings.Join([]string{token0, token1}, "-")
	}

	return strings.Join([]string{token1, token0}, "-")
}
