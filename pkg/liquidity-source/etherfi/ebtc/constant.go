package etherfiebtc

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "etherfi-ebtc"

	vaultDecimals = 8
)

var (
	oneShare        = big256.TenPowInt(vaultDecimals)
	maxSharePremium = uint256.NewInt(1000)

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
