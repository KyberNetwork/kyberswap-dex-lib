package dynamicfee

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
)

// CLHookFeeConfig https://github.com/pancakeswap/pancake-frontend/blob/c9d4f1fb71f122aa7e3d735b6f0e5475953c8d2b/packages/infinity-sdk/src/constants/hooksList/bsc.ts#L6-L25
var CLHookFeeConfig = map[common.Address]uint32{
	common.HexToAddress("0x32c59d556b16db81dfc32525efb3cb257f7e493d"): 500,
}

func CLHookAddress() []common.Address {
	return lo.Keys(CLHookFeeConfig)
}

func GetDefaultFee(hookAddress common.Address) uint32 {
	if fee, ok := CLHookFeeConfig[hookAddress]; ok {
		return fee
	}
	return shared.MAX_FEE_PIPS
}
