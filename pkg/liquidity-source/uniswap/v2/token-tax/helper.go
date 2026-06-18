package tokentax

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func FindPairedToken(pool entity.Pool, baseTokens map[string]struct{}) string {
	if len(pool.Tokens) != 2 {
		return ""
	}
	for i, token := range pool.Tokens {
		if _, ok := baseTokens[strings.ToLower(token.Address)]; ok {
			return strings.ToLower(pool.Tokens[1-i].Address)
		}
	}
	return ""
}
