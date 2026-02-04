package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

func anyPoolKey(
	token0 string,
	token1 string,
	extension string,
	fee uint64,
	poolTypeConfig pools.PoolTypeConfig,
) pools.AnyPoolKey {
	return pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
		common.HexToAddress(token0),
		common.HexToAddress(token1),
		pools.NewPoolConfig(common.HexToAddress(extension), fee, poolTypeConfig),
	)}
}
