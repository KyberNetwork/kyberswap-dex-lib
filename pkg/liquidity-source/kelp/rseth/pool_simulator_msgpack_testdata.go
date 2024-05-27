package rseth

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xa1290d69c65a6fe4df752f95823fae25cb99e5a7", // rsETH
						"0xa35b1b31ce002fbf2058d22f30f95d405200a15b", // ETHx
					},
				},
			},

			minAmountToDeposit:  bignumber.NewBig("100000000000000"),
			totalDepositByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("802460400000000000000")},
			depositLimitByAsset: map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("4197539600000000000000")},
			priceByAsset:        map[string]*big.Int{"0xa35b1b31ce002fbf2058d22f30f95d405200a15b": bignumber.NewBig("1015786347348446492")},
			rsETHPrice:          bignumber.NewBig("1000000000000000000"),
		},
	}
}
