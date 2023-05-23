package platypus

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func getOracleType(oraclePrice string) string {
	switch oraclePrice {
	case addressZero:
		return oracleTypeNone
	case addressStakedAvax:
		return oracleTypeStakedAvax
	default:
		return oracleTypeChainlink
	}
}

func getPoolTypeByPriceOracle(oraclePrice string) string {
	switch oraclePrice {
	case addressZero:
		return poolTypePlatypusPure
	case addressStakedAvax:
		return poolTypePlatypusAvax
	default:
		return poolTypePlatypusBase
	}
}

func newPoolTokens(tokens []common.Address) []*entity.PoolToken {
	poolTokens := make([]*entity.PoolToken, 0, len(tokens))
	for _, token := range tokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   strings.ToLower(token.Hex()),
			Swappable: true,
		})
	}

	return poolTokens
}
