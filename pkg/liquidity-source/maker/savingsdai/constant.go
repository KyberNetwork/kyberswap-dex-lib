package savingsdai

import (
	"errors"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "maker-savingsdai"

	potMethodRHO = "rho"
	potMethodCHI = "chi"

	savingsMethodTotalAssets = "totalAssets"
	savingsMethodTotalSupply = "totalSupply"

	Blocktime = 12
)

var (
	RAY = big256.TenPowInt(27)
)

var (
	savingsDAIDefaultGas = Gas{
		Deposit: 160000,
		Redeem:  146000,
	}

	savingsUSDSDefaultGas = Gas{
		Deposit: 137000,
		Redeem:  145000,
	}
)

var (
	ErrInvalidToken = errors.New("invalid token")
)
