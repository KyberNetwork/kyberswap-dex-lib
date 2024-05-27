package eeth

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
					Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x35fa164735182de50811e8e2e824cfb9b6118ac2"},
				},
			},
			totalPooledEther: bignumber.NewBig("478349632983976798301885"),
			totalShares:      bignumber.NewBig("463434527744908632824686"),
		},
	}
}
