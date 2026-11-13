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

	// MEVCapture v2.1: Ekubo's calibration of the forwarded swap measured against the same swaps on pools without
	// extension (per forwarded swap 8,250 plus a cold extension access 2,500, per Core call 12,350, ...), applied on
	// top of the concentrated pool base cost above in place of the 12,400 base of a plain swap in that calibration.
	BaseGasCostOfOneMevCaptureV21Swap        = BaseGasCostOfOneConcentratedLiquiditySwap - 12_400 + 8_250 + 2_500
	GasCostOfOneMevCaptureV21CoreCall        = 12_350
	GasCostOfOneMevCaptureV21ExtraCoreStep   = 2_850
	GasCostOfOneMevCaptureV21TickCrossed     = 13_150
	GasCostOfOneMevCaptureV21FeeAccumulation = 7_950
	GasCostOfMevCaptureV21AnchorUpdate       = 4_650

	// BoostedFees
	ExtraBaseGasCostOfOneBoostedFeesSwap   = 2_743
	GasCostOfExecutingVirtualDonations     = 6_814
	GasCostOfCrossingOneVirtualDonateDelta = 4_271
	GasCostOfBoostedFeesFeeAccumulation    = 19_279

	// Ve33
	ExtraBaseGasCostOfOneVe33Swap = 30_000
)
