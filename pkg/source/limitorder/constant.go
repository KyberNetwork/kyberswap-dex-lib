package limitorder

const (
	DexTypeLimitOrder = "limit-order"

	PrefixLimitOrderPoolID              = "limit_order_pool"
	SeparationCharacterLimitOrderPoolID = "_"

	// Currently, the total of TVL/reserveUsd in the limit order pool will be very small compared with other pools. So it will be filtered in choosing pools process
	// We will use big hardcode number to push it into eligible pools for findRoute algorithm.
	// TODO: when we has correct formula that pool's reserve can be eligible pools.
	limitOrderPoolReserve    = "10000000000000000000"
	LimitOrderPoolReserveUSD = 1000000000
)
