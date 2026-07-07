package valantisstex

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeValantisStex
)

var (
	maxSwapFeeBips = u256.UBasisPoint
)

var (
	ErrInvalidToken                      = errors.New("invalid token")
	ErrZeroSwap                          = errors.New("zero swap")
	ErrSovereignPoolSwapExcessiveSwapFee = errors.New("swap excessive swap fee")
	ErrInsufficientReserve               = errors.New("insufficient reserve")
	ErrInvalidGasConfig                  = errors.New("invalid gas config")
)
