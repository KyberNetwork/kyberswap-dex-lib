package stablemetang

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/holiman/uint256"
)

type (
	StaticExtra struct {
		APrecision          *uint256.Int
		OffpegFeeMultiplier *uint256.Int
		// which coins are originally native (before being converted to wrapped)
		IsNativeCoins []bool
		BasePool      string
	}

	Extra struct {
		InitialA     *uint256.Int
		FutureA      *uint256.Int
		InitialATime int64
		FutureATime  int64
		SwapFee      *uint256.Int
		AdminFee     *uint256.Int

		RateMultipliers []uint256.Int `json:",omitempty"`
	}

	MetaPoolSwapInfo struct {
		TokenInIndex  int
		TokenOutIndex int
		AmountIn      uint256.Int
		AmountOut     uint256.Int
		AdminFee      uint256.Int
	}

	BasePoolAddLiquidityInfo struct {
		Amounts    [shared.MaxTokenCount]uint256.Int
		MintAmount uint256.Int
		FeeAmounts [shared.MaxTokenCount]uint256.Int
	}

	BasePoolWithdrawInfo struct {
		TokenAmount uint256.Int
		TokenIndex  int
		Dy          uint256.Int
		DyFee       uint256.Int
	}

	SwapInfo struct {
		AddLiquidity *BasePoolAddLiquidityInfo
		Meta         *MetaPoolSwapInfo
		Withdraw     *BasePoolWithdrawInfo
	}
)
