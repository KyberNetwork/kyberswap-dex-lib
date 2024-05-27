package unieth

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{WETH, UNIETH},
				},
			},
			paused:         false,
			totalSupply:    bignumber.NewBig("40654517980271452478787"),
			currentReserve: bignumber.NewBig("43102498463014375406128"),
		},
	}
}
