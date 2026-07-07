package platypus

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

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
		return PoolTypePlatypusPure
	case addressStakedAvax:
		return PoolTypePlatypusAvax
	default:
		return PoolTypePlatypusBase
	}
}

func newPoolTokens(tokens []common.Address) []*entity.PoolToken {
	poolTokens := make([]*entity.PoolToken, 0, len(tokens))
	for _, token := range tokens {
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   hexutil.Encode(token[:]),
			Swappable: true,
		})
	}

	return poolTokens
}
