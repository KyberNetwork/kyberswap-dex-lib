package dexT1

const (
	DexType = "fluid-dex-t1"
)

const (
	// DexReservesResolver methods
	DRRMethodGetAllPoolsReserves = "getAllPoolsReserves"
	DRRMethodGetPoolReserves     = "getPoolReserves"

	// ERC20 Token methods
	TokenMethodDecimals = "decimals"
)

const DexAmountsDecimals int64 = 12

const FeePercentPrecision int64 = 1e4
const Fee100PercentPrecision int64 = 1e6

const NativeETH string = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
