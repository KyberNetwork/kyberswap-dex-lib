package pufeth

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{PUFETH, STETH, WSTETH},
				},
			},
			totalSupply: number.NewUint256("379989503452489947895013"),
			totalAssets: number.NewUint256("382649667359278267721330"),
		},
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{PUFETH, STETH, WSTETH},
				},
			},
			totalSupply:      number.NewUint256("379677392580527064900714"),
			totalAssets:      number.NewUint256("382335371516233372457736"),
			totalPooledEther: number.NewUint256("9408886941382666867434878"),
			totalShares:      number.NewUint256("8085737150987915500442326"),
		},
	}
}
