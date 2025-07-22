package aegis

import (
	"github.com/ethereum/go-ethereum/common"

	uniswapv4types "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/types"
)

type StaticExtraAegis struct {
	uniswapv4types.StaticExtra
	DynamicFeeManagerAddress common.Address `json:"dFM"`
	PolicyManagerAddress     common.Address `json:"pM"`
}

type ExtraAegis struct {
	uniswapv4types.Extra
	BaseFee        uint64 `json:"baseFee"`
	SurgeFee       uint64 `json:"surgeFee"`
	ManualFee      uint64 `json:"manualFee"`
	ManualFeeIsSet bool   `json:"manualFeeIsSet"`
	DynamicFee     uint64 `json:"dynamicFee"`
	PoolPOLShare   uint64 `json:"poolPOLShare"`
}
