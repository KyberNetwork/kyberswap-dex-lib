package lglclob

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "lgl-clob"

	maxPriceLevels = 48

	safetyBuffer        = 0.69420
	priceLimitPrecision = 6
)

var (
	bMaxPriceLevels       = big.NewInt(maxPriceLevels)
	uPriceLimitMultiplier = new(uint256.Int).AddUint64(big256.UBasisPoint, 12)

	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrEmptyOrders          = errors.New("empty orders")
	ErrExceededSafetyBuffer = errors.New("exceed safety buffer")
)
