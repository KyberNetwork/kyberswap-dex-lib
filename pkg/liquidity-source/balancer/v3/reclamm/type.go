package reclamm

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
)

type Extra struct {
	*shared.Extra
	LastVirtualBalances       []*uint256.Int `json:"lastVirtualBalances"`
	DailyPriceShiftBase       *uint256.Int   `json:"dailyPriceShiftBase"`
	LastTimestamp             *uint256.Int   `json:"lastTimestamp"`
	CurrentTimestamp          *uint256.Int   `json:"currentTimestamp"`
	CenterednessMargin        *uint256.Int   `json:"centerednessMargin"`
	StartFourthRootPriceRatio *uint256.Int   `json:"startFourthRootPriceRatio"`
	EndFourthRootPriceRatio   *uint256.Int   `json:"endFourthRootPriceRatio"`
	PriceRatioUpdateStartTime *uint256.Int   `json:"priceRatioUpdateStartTime"`
	PriceRatioUpdateEndTime   *uint256.Int   `json:"priceRatioUpdateEndTime"`
}

type RpcResult struct {
	shared.RpcResult
	DynamicDataRpc
}

type DynamicDataRpc struct {
	Data struct {
		BalancesLiveScaled18        []*big.Int
		TokenRates                  []*big.Int
		StaticSwapFeePercentage     *big.Int
		TotalSupply                 *big.Int
		LastTimestamp               *big.Int
		LastVirtualBalances         []*big.Int
		DailyPriceShiftExponent     *big.Int
		DailyPriceShiftBase         *big.Int
		CenterednessMargin          *big.Int
		CurrentPriceRatio           *big.Int
		CurrentFourthRootPriceRatio *big.Int
		StartFourthRootPriceRatio   *big.Int
		EndFourthRootPriceRatio     *big.Int
		PriceRatioUpdateStartTime   uint32
		PriceRatioUpdateEndTime     uint32
		IsPoolInitialized           bool
		IsPoolPaused                bool
		IsPoolInRecoveryMode        bool
	}
}
