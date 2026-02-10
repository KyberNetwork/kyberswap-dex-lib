package quoting

const (
	GasCostOfOneColdSload = 2_100

	BaseGasCost = 56_502

	BaseGasFullRangeSwap             = 15_678
	BaseGasStableswapSwap            = 17_184
	BaseGasConcentratedLiquiditySwap = 19_175
	ExtraBaseGasMevCaptureSwap       = 9_178

	GasInitializedTickCrossed     = 16_420
	GasTickSpacingCrossed         = 2_507
	GasUpdatingOracleSnapshot     = 9_828
	GasVirtualOrderDelta          = 25_000
	GasExecutingVirtualOrders     = 25_502
	GasAccumulatingMevCaptureFees = 42_123

	ExtraBaseGasCostOfOneBoostedFeesSwap   = 2_743
	GasCostOfExecutingVirtualDonations     = 6_814
	GasCostOfCrossingOneVirtualDonateDelta = 4_271
	GasCostOfBoostedFeesFeeAccumulation    = 19_279
)
