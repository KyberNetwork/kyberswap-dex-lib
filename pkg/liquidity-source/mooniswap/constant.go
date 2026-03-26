package mooniswap

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType          = valueobject.ExchangeMooniswap
	defaultGas int64 = 200000

	poolMethodFee                   = "fee"
	poolMethodSlippageFee           = "slippageFee"
	poolMethodToken0                = "token0"
	poolMethodToken1                = "token1"
	poolMethodGetBalanceForAddition = "getBalanceForAddition"
	poolMethodGetBalanceForRemoval  = "getBalanceForRemoval"
	poolMethodGetReturn             = "getReturn"

	factoryMethodGetAllPools = "getAllPools"
)

var (
	uFeeDenominator = big256.TenPow(18)
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrZeroAmount            = errors.New("zero amount")
)
