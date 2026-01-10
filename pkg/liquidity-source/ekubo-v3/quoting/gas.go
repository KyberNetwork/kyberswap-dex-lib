package quoting

// TODO
const (
	BaseGasCost = 81_473

	BaseGasFullRangeSwap             = 17_351
	BaseGasStableswapSwap            = 18_000 // TODO
	BaseGasConcentratedLiquiditySwap = 21_827
	ExtraBaseGasMevCaptureSwap       = 9_178

	GasInitializedTickCrossed     = 20_000
	GasTickSpacingCrossed         = 2_507
	GasUpdatingOracleSnapshot     = 16_821
	GasVirtualOrderDelta          = 25_000
	GasExecutingVirtualOrders     = 25_502
	GasAccumulatingMevCaptureFees = 11_264
)
