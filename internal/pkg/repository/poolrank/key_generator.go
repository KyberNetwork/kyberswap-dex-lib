package poolrank

import (
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type keyGenerator struct {
	prefix string
}

func NewKeyGenerator(prefix string) *keyGenerator {
	return &keyGenerator{
		prefix: prefix,
	}
}

// directPairKey generates key of direct pairs
func (g *keyGenerator) directPairKey(sortBy, token0, token1 string) string {
	return utils.Join(g.prefix, sortBy, g.joinTokens(token0, token1))
}

// whitelistToWhitelistPairKey generates key of pairs between whitelisted tokens
func (g *keyGenerator) whitelistToWhitelistPairKey(sortBy string) string {
	return utils.Join(g.prefix, sortBy, KeyWhitelist)
}

// whitelistToTokenPairKey generates key of pairs between a whitelisted token and a non-whitelisted token
func (g *keyGenerator) whitelistToTokenPairKey(sortBy, token string) string {
	return utils.Join(g.prefix, sortBy, KeyWhitelist, token)
}

func (g *keyGenerator) joinTokens(token0, token1 string) string {
	if token0 > token1 {
		return fmt.Sprintf("%s-%s", token0, token1)
	}

	return fmt.Sprintf("%s-%s", token1, token0)
}
