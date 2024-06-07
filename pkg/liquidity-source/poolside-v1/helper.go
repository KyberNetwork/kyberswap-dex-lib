package poolsidev1

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func newPoolData(p poolDataResp) *poolData {
	var (
		poolTokens      = make([]*entity.PoolToken, len(p.Data.ListedTokens))
		poolReserves    = make([]string, len(p.Data.ListedTokens))
		lpTokenBalances = make(map[string]*big.Int)
	)

	for i, token := range p.Data.ListedTokens {
		t := strings.ToLower(common.BytesToAddress(token[:]).Hex())
		poolTokens[i] = &entity.PoolToken{
			Address:   t,
			Weight:    defaultWeight,
			Swappable: true,
		}
		poolReserves[i] = p.Data.Reserves[i].String()
		lpTokenBalances[t] = new(big.Int).Sub(maxUint128, p.Data.MintedLPTokens[i])
	}

	fee1e8 := new(big.Int).SetBytes(p.Data.PoolParams[:32])
	amp := new(big.Int).SetBytes(p.Data.PoolParams[32:])

	return &poolData{
		Tokens:          poolTokens,
		PoolReserves:    poolReserves,
		Amp:             amp,
		Fee1e18:         fee1e8,
		LpTokenBalances: lpTokenBalances,
	}
}
