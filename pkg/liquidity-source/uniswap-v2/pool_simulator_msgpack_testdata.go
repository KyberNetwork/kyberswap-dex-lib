package uniswapv2

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Address:  "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
					Tokens:   []string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
					Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576")},
				},
			},
			fee:          number.NewUint256("3"),
			feePrecision: number.NewUint256("1000"),
		},
	}
}
