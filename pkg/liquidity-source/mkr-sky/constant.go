package mkr_sky

import "math/big"

const (
	DexType = "mkr-sky"

	OneWayPoolAddress = "0xa1ea1ba18e88c381c724a75f23a130420c403f9a"
	MkrAddress        = "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"
	SkyAddress        = "0x56072c95faa701256059aa122697b133aded9279"

	wad                   = 1e18
	defaultReserves       = "10000000000000000000"
	DefaultGas      int64 = 60000
)

var (
	WAD = big.NewInt(1e18) // 10**18
)
