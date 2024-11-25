package lo1inch

import "math/big"

const (
	DexType = "lo1inch"

	PoolIDPrefix    = "lo1inch"
	PoolIDSeparator = "_"

	// Currently, the total of TVL/reserveUsd in the limit order pool will be very small compared with other pools. So it will be filtered in choosing pools process
	// We will use big hardcode number to push it into eligible pools for findRoute algorithm.
	// TODO: when we has correct formula that pool's reserve can be eligible pools.
	limitOrderPoolReserve    = "10000000000000000000"
	LimitOrderPoolReserveUSD = 1000000000
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
