package limitorder

import (
	"math/big"
)

var (
	// BasGas is base gas to executor a tx for LO.
	BaseGas = 90000

	// GasPerOrderExecutor is gas for executing an order.
	GasPerOrderExecutor = 11100
	// GasPerOrderRouter need to burn when sending in call data.
	GasPerOrderRouter = 12208
	// FallbackPercentageOfTotalMakingAmount is fallback percentage of total remain making amount with amount out.
	// total remain making amount = total remain making amount(filled orders) + total remain making amount(fallback orders)
	FallbackPercentageOfTotalMakingAmount = big.NewFloat(1.3)
)
