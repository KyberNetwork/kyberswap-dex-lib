package cmeth

import (
	"errors"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "cmeth"

	vaultDecimals = 18
)

var (
	oneShare = big256.TenPow(vaultDecimals)

	defaultReserves = "10000000000000000000000"

	defaultGas = Gas{
		Deposit: 90000,
	}
)

const (
	tellerMethodIsPaused  = "isPaused"
	tellerMethodIsSupported = "isSupported"
	accountantMethodAccountantState  = "accountantState"
	accountantMethodRateProviderData = "rateProviderData"
)

var (
	ErrTellerPaused            = errors.New("teller with multi asset support: paused")
	ErrTellerAssetNotSupported = errors.New("teller with multi asset support: asset not supported")
	ErrTellerMinimumMintNotMet = errors.New("teller with multi asset support: minimum mint not met")
	ErrTellerZeroAssets        = errors.New("teller with multi asset support: zero assets")
	ErrAccountantPaused        = errors.New("accountant with rate providers: paused")
	ErrMulDivOverflow          = errors.New("mul div overflow")
)
