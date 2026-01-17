package quoting

// TODO
const (
	BaseGasCost = 27_326

	BaseGasFullRangeSwap             = 16_079
	BaseGasStableswapSwap            = 18_000 // TODO
	BaseGasConcentratedLiquiditySwap = 19_360
	ExtraBaseGasMevCaptureSwap       = 9_178

	GasInitializedTickCrossed     = 20_000
	GasTickSpacingCrossed         = 2_507
	GasUpdatingOracleSnapshot     = 16_821
	GasVirtualOrderDelta          = 25_000
	GasExecutingVirtualOrders     = 25_502
	GasAccumulatingMevCaptureFees = 11_264
)
