package twocryptong

import "github.com/holiman/uint256"

type (
	StaticExtra struct {
		// which coins are originally native (before being converted to wrapped)
		IsNativeCoins []bool
	}

	Extra struct {
		InitialA          *uint256.Int
		InitialGamma      *uint256.Int
		InitialAGammaTime int64 `json:",omitempty"`
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

		// Version is the pool's version() ("v2.0.0", "v2.1.0d", ...). v2.1.0d rebalances by its own rule (see
		// tweakPriceV21d) and needs the fields below: LastTimestamp is last_timestamp (one rebalance per block), and
		// the donation fields are the donation_* views, which count unlocked donation shares in the rebalance test.
		Version                    string       `json:",omitempty"`
		LastTimestamp              int64        `json:",omitempty"`
		DonationShares             *uint256.Int `json:",omitempty"`
		DonationDuration           *uint256.Int `json:",omitempty"`
		LastDonationReleaseTs      int64        `json:",omitempty"`
		DonationProtectionExpiryTs int64        `json:",omitempty"`
		DonationProtectionPeriod   *uint256.Int `json:",omitempty"`

		// v3.0.0 only: lp_xcp_profit, reserved_profit_fraction and admin_fee (the rebalance threshold and the admin
		// share of each fee), [adjustment_step_min, adjustment_step_max] from packed_rebalancing_params, and the
		// state of a YBTwocryptoPolicy, whose get_price_scale is the rebalance target.
		LpXcpProfit            *uint256.Int          `json:",omitempty"`
		ReservedProfitFraction *uint256.Int          `json:",omitempty"`
		AdminFee               *uint256.Int          `json:",omitempty"`
		AdjustmentStepMin      *uint256.Int          `json:",omitempty"`
		AdjustmentStepMax      *uint256.Int          `json:",omitempty"`
		TwocryptoPolicy        *TwocryptoPolicyState `json:",omitempty"`

		// PolicyAddress is the pool's POLICY contract (v3.0.0), lower-case hex, empty when the pool has none or no
		// view. A policy can set the fee and the price-scale target (see nativeFeePolicies and Policy).
		PolicyAddress string `json:",omitempty"`

		FeeGamma *uint256.Int
		MidFee   *uint256.Int
		OutFee   *uint256.Int

		LpSupply           *uint256.Int
		XcpProfit          *uint256.Int
		VirtualPrice       *uint256.Int
		AllowedExtraProfit *uint256.Int
		AdjustmentStep     *uint256.Int
		UseCustomMath      bool `json:",omitempty"`
		// Twocrypto v3 (no allowed_extra_profit): AllowedExtraProfit is nil.
		// Policy prices swaps through YBOraclizedPolicy.get_fee (report-aware discount).
		Policy *PolicyState `json:",omitempty"`
		// PolicyFee is a flat fee override kept for previously stored pool data; the trackers no
		// longer set it (an unlisted policy errors in FetchPolicy). Both nil: the pool's native fee.
		PolicyFee *uint256.Int `json:",omitempty"`
	}

	// TwocryptoPolicyState is a YBTwocryptoPolicy's state() and immutables.
	TwocryptoPolicyState struct {
		LastUpdateTs                      int64
		LastPrices, FastEma, SlowEma      *uint256.Int
		PriceScale                        *uint256.Int
		FastHalfLife, SlowHalfLife, Kappa *uint256.Int
		Deadband, MinCap, MaxCap          *uint256.Int
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

		// v2.1.0d only (see tweakPriceV21d): the supply and donation state after a rebalance's burn, and the
		// block's timestamp, which allows no second rebalance in the block.
		LpSupply              uint256.Int
		DonationShares        uint256.Int
		LastDonationReleaseTs int64
		LastTimestamp         int64

		// v3.0.0 only (see tweakPriceV3): the admin share of the fee, which leaves balances[j], lp_xcp_profit, the
		// policy's state after update_pool_state, and whether the state after the trade is known (V3Exact).
		AdminFee        uint256.Int
		LpXcpProfit     uint256.Int
		TwocryptoPolicy *TwocryptoPolicyState
		V3Exact         bool
	}
)
