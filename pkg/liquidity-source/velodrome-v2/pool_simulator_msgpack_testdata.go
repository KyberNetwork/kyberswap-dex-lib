package velodromev2

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
					Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
					Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
				},
			},
			isPaused:     false,
			stable:       true,
			decimals0:    number.NewUint256("1000000000000000000"),
			decimals1:    number.NewUint256("1000000000000000000"),
			fee:          uint256.NewInt(5),
			feePrecision: uint256.NewInt(10000),
		},
	}
}
