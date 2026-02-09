package someswapv2

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeSomeSwapV2

const (
	poolsEndpoint      = "/api/amm/pools/v2"
	dynamicFeeEndpoint = "/api/amm/dynamic-fee/{pool-address}"

	defaultGas = 80000
)

var bpsDen = u256.TenPow(9)
