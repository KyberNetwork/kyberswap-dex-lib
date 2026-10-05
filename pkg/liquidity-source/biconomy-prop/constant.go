package biconomyprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeBiconomyProp

	methodGetPairs = "getPairs"
	methodMakers   = "makers"
	methodBoard    = "board"
	methodFeeBps   = "feeBps"

	bpsDenominator = 10_000

	// Venue swap gas: one maker fill, plus each further maker the order reaches.
	defaultGas   = 280_000
	perMemberGas = 65_000
)

var (
	ErrInvalidToken          = errors.New("biconomy-prop: tokenIn/tokenOut is not the pool pair")
	ErrZeroAmountIn          = errors.New("biconomy-prop: zero amountIn")
	ErrInsufficientLiquidity = errors.New("biconomy-prop: amountIn exceeds live board depth")
	ErrZeroAmountOut         = errors.New("biconomy-prop: zero amountOut")
	ErrOverflow              = errors.New("biconomy-prop: uint256 overflow")
)
