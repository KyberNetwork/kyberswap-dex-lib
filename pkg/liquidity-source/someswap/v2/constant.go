package someswapv2

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeSomeSwapV2

const (
	poolsEndpoint = "/api/amm/pools/v2"

	defaultGas = 80000
)

var (
	feeDen    = u256.TenPow(6)
	weightDen = u256.TenPow(9)
)
