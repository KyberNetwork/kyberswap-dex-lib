package biconomyprop

import (
	"errors"
	"time"

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

	// defaultFeedMaxAge bounds how old a feed snapshot may be. The feed sends one per Base block
	// (2s) while subscribed, so an older one means the feed is behind and the tracker reads
	// the venue on-chain instead.
	defaultFeedMaxAge = 5 * time.Second
)

var (
	ErrInvalidToken          = errors.New("biconomy-prop: tokenIn/tokenOut is not the pool pair")
	ErrZeroAmountIn          = errors.New("biconomy-prop: zero amountIn")
	ErrInsufficientLiquidity = errors.New("biconomy-prop: amountIn exceeds live board depth")
	ErrZeroAmountOut         = errors.New("biconomy-prop: zero amountOut")
	ErrOverflow              = errors.New("biconomy-prop: uint256 overflow")
)
