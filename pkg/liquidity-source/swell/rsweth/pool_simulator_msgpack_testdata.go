package rsweth

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{common.WETH, common.RSWETH},
				},
			},
			paused:          false,
			ethToRswETHRate: bignumber.NewBig("995131146747098421"),
		},
	}
}
