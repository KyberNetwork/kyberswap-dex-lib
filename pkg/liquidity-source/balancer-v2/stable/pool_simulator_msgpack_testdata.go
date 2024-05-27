package stable

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: []*big.Int{
						bignumber.NewBig("9999991000000000000000"),
						bignumber.NewBig("9999991000000000005613"),
						bignumber.NewBig("13288977911102200123456"),
					},
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(50000000000000),
			amp:               uint256.NewInt(1390000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1), uint256.NewInt(100)},

			poolType:    poolTypeStable,
			poolTypeVer: 1,
		},
	}
}
