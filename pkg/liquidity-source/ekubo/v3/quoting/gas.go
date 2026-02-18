package quoting

const (
	BaseGasCost           = 56_502
	GasCostOfOneColdSload = 2_100

	// Full range
	GasCostOfOneFullRangeSwap = 15_678

	// Stableswap
	GasCostOfOneStableswapSwap = 17_184

	// Concentrated
	BaseGasCostOfOneConcentratedLiquiditySwap = 19_175
	GasCostOfOneInitializedTickCrossed        = 14_259 // The first crossed tick is more expensive because of the `sload` of the output token fees per liquidity but we can't capture that. This estimate is for the first tick.
	GasCostOfOneExtraTickBitmapSload          = 2_000  // Difference between an sload on a warm vs a cold slot
	GasCostOfOneExtraConcentratedMathRound    = 4_076

	// Oracle
	ExtraBaseGasCostOfOneOracleSwap = -1_801 // The negative costs come from savings compared to a full range swap which usually touches fee-related storage slots
	GasCostOfUpdatingOracleSnapshot = 9_709

	// TWAMM
	ExtraBaseGasCostOfOneTwammSwap        = 5_302
	GasCostOfExecutingVirtualOrders       = 20_554
	GasCostOfCrossingOneVirtualOrderDelta = 19_980

	// MEVCapture
	ExtraBaseGasCostOfOneMevCaptureSwap = 15_840
	GasCostOfOneMevCaptureStateUpdate   = 16_418

	// BoostedFees
	ExtraBaseGasCostOfOneBoostedFeesSwap   = 2_743
	GasCostOfExecutingVirtualDonations     = 6_814
	GasCostOfCrossingOneVirtualDonateDelta = 4_271
	GasCostOfBoostedFeesFeeAccumulation    = 19_279
)
