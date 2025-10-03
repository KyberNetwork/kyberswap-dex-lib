package cusd

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
)

const (
	DexType = "cusd"

	capTokenAssetsMethod        = "assets"
	capTokenWhitelistedMethod   = "whitelisted"
	capTokenTotalSuppliesMethod = "totalSupplies"
	capTokenGetFeeDataMethod    = "getFeeData"

	oracleGetPriceMethod = "getPrice"

	defaultMintGas int64 = 0
	defaultBurnGas int64 = 0
)

var (
	rayPrecision   = number.TenPow(27)
	sharePrecision = number.TenPow(33)
)

var (
	ErrInvalidToken = errors.New("invalid token")
)
