package quoting

const (
	BaseGasCost = 29_000

	BaseGasCostOfOneFullRangeSwap              = 20_000
	BaseGasCostOfOneConcentratedLiquidtitySwap = 24_000
	ExtraBaseGasCostOfMevResistSwap            = 17_600

	GasCostOfOneInitializedTickCrossed = 20_000
	GasCostOfOneTickSpacingCrossed     = 4_000
	GasCostOfUpdatingOracleSnapshot    = 15_000
	GasCostOfOneVirtualOrderDelta      = 25_000
	GasCostOfExecutingVirtualOrders    = 15_000
	GasCostOfAccumulatingMevResistFees = 14_900
)
