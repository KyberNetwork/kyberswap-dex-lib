package weighted

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xac3E018457B222d93114458476f3E3416Abbe38F",
						"0xae78736Cd615f374D3085123A210448E74Fc6393",
						"0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
					},
					Reserves: []*big.Int{
						big.NewInt(331125),
						big.NewInt(320633),
						big.NewInt(348846),
					},
				},
			},

			swapFeePercentage: uint256.NewInt(3000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333400000000000000),
			},
			totalAmountsIn: []*uint256.Int{uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0)},
			scaledMaxTotalAmountsIn: []*uint256.Int{
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
			poolTypeVer: 3,
		},
	}
}
