package cusd

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
)

const (
	DexType = "cusd"

	capTokenAssetsMethod           = "assets"
	capTokenWhitelistedMethod      = "whitelisted"
	capTokenTotalSuppliesMethod    = "totalSupplies"
	capTokenGetFeeDataMethod       = "getFeeData"
	capTokenPausedMethod           = "paused"
	capTokenAvailableBalanceMethod = "availableBalance"

	oracleGetPriceMethod = "getPrice"

	pausablePausedMethod = "paused"

	defaultMintGas int64 = 0
	defaultBurnGas int64 = 0
)

var (
	rayPrecision = number.TenPow(27)
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInsufficientReserves = errors.New("insufficient reserves")
	ErrContractPaused       = errors.New("contract paused")
	ErrAssetPaused          = errors.New("asset paused")
	ErrAssetNotSupported    = errors.New("asset not supported")
)
