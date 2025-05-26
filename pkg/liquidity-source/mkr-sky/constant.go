package mkr_sky

import "math/big"

const (
	DexType = "mkr-sky"

	MkrAddress = "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"
	SkyAddress = "0x56072c95faa701256059aa122697b133aded9279"

	defaultReserves       = "10000000000000000000"
	DefaultGas      int64 = 60000
)

var (
	WAD = big.NewInt(1e18) // 10**18
)
