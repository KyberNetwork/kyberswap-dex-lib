package tessera

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "tessera"

	defaultGas = 400000
)

var (
	// Support up to 65% capacity of order book to avoid revert due to on-chain state changes
	maxOrderbookFillFactorBPS = uint256.NewInt(6500)
	ErrInvalidToken           = errors.New("invalid token")
	ErrTradingDisabled        = errors.New("trading disabled")
	ErrNotInitialised         = errors.New("pool not initialised")
	ErrInvalidRate            = errors.New("invalid rate")
	ErrInvalidSwapRate        = errors.New("invalid swap rate")
	ErrSwapReverted           = errors.New("swap would revert")
	ErrInternal               = errors.New("internal error")
)
