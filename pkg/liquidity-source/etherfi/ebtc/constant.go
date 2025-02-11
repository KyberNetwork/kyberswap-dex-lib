package etherfiebtc

import (
	"errors"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "etherfi-ebtc"

	vaultDecimals = 8
)

var (
	oneShare = big256.TenPowInt(vaultDecimals)

	defaultReserves = "10000000000"

	defaultGas = Gas{
		Deposit: 90000,
	}
)

const (
	tellerMethodIsPaused             = "isPaused"
	tellerMethodAssetData            = "assetData"
	tellerMethodShareLockPeriod      = "shareLockPeriod"
	accountantMethodAccountantState  = "accountantState"
	accountantMethodRateProviderData = "rateProviderData"
)

var (
	ErrTellerPaused            = errors.New("teller with multi asset support: paused")
	ErrTellerAssetNotSupported = errors.New("teller with multi asset support: asset not supported")
	ErrTellerMinimumMintNotMet = errors.New("teller with multi asset support: minimum mint not met")
	ErrTellerZeroAssets        = errors.New("teller with multi asset support: zero assets")
	ErrTellerSharesAreLocked   = errors.New("teller with multi asset support: shares are locked")
	ErrAccountantPaused        = errors.New("accountant with rate providers: paused")
	ErrMulDivOverflow          = errors.New("mul div overflow")
)
