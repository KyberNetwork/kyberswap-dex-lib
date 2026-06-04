package liquidcore

import (
	"errors"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeLiquidCore

	defaultGas = 139441
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrInsufficientReserve = errors.New("insufficient reserve")
	ErrZeroAmount          = errors.New("zero amount")
	ErrSpotPriceZero       = errors.New("spot price is zero")

	uScale    = uint256.NewInt(100_000)
	uBalanced = uint256.NewInt(50_000)
	uBaseFee  = uint256.NewInt(25)
	uMinFee   = uint256.NewInt(1)
	u6969     = uint256.NewInt(6969)

	u1e6  = u256.TenPow(6)
	u1e10 = u256.TenPow(10)
	u1e18 = u256.TenPow(18)
)
