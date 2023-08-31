package kyberpmm

type SwapDirection uint8

const (
	DexTypeKyberPMM = "kyber-pmm"

	PoolIDPrefix    = "kyber_pmm"
	PoolIDSeparator = "_"

	// Currently, the total of TVL/reserveUsd in the Kyber PMM pool will be very small compared with other pools. So it will be filtered out in choosing pools process
	// We will use big hardcode number to push it into eligible pools for findRoute algorithm.
	// TODO: update this when we have a correct formula for Kyber PMM pools to be eligible pools.
	poolReserve = "1000000000000000000000000" // 1e6 * 1e18
)

const (
	// SwapDirectionBaseToQuote is the direction of swap from base to quote
	SwapDirectionBaseToQuote SwapDirection = iota
	// SwapDirectionQuoteToBase is the direction of swap from quote to base
	SwapDirectionQuoteToBase
)

var (
	DefaultGas = Gas{Swap: 100000}
)
