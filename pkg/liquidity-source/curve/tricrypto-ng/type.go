package tricryptong

import "github.com/holiman/uint256"

type (
	StaticExtra struct {
		// which coins are originally native (before being converted to wrapped)
		IsNativeCoins []bool
	}

	Extra struct {
		InitialA          *uint256.Int
		InitialGamma      *uint256.Int
		InitialAGammaTime int64
		FutureA           *uint256.Int
		FutureGamma       *uint256.Int
		FutureAGammaTime  int64

		D *uint256.Int

		PriceScale  []uint256.Int
		PriceOracle []uint256.Int
		LastPrices  []uint256.Int

		FeeGamma *uint256.Int
		MidFee   *uint256.Int
		OutFee   *uint256.Int

		LpSupply           *uint256.Int
		XcpProfit          *uint256.Int
		VirtualPrice       *uint256.Int
		AllowedExtraProfit *uint256.Int
		AdjustmentStep     *uint256.Int
	}

	SwapInfo struct {
		K0           uint256.Int
		Xp           [NumTokens]uint256.Int
		LastPrices   [NumTokens - 1]uint256.Int
		PriceScale   [NumTokens - 1]uint256.Int
		XcpProfit    uint256.Int
		D            uint256.Int
		VirtualPrice uint256.Int
	}
)
