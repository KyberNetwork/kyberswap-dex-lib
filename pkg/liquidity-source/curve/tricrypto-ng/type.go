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

		// PriceOracleTimestamp is when the tracker read PriceOracle, in unix seconds, and MaTime is the EMA's
		// time constant (packed_rebalancing_params; the ma_time() view reports it times ln 2). The contract's
		// price_oracle() keeps moving toward last_prices until the next trade, so the simulator carries
		// PriceOracle forward to the swap's time (see priceOracleAt). The read's block can be up to one block
		// older than PriceOracleTimestamp, which leaves the carried value at most one block short.
		PriceOracleTimestamp int64        `json:",omitempty"`
		MaTime               *uint256.Int `json:",omitempty"`

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

		// PriceOracle is price_oracle at PriceOracleTimestamp, the time the swap is quoted at.
		PriceOracle          [NumTokens - 1]uint256.Int
		PriceOracleTimestamp int64
	}
)
