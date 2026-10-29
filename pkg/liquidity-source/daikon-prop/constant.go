package daikonprop

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeDaikonProp

	methodManager        = "manager"
	methodVault          = "vault"
	methodPairCount      = "pairCount"
	methodAllPairs       = "allPairs"
	methodGetMarketState = "getMarketState"

	// maxFeePpm is the 5% cap the pair applies to swapData fees; larger values are cut.
	maxFeePpm = 50_000

	defaultGas int64 = 150_000
)

var (
	ErrInvalidToken   = errors.New("daikon-prop: tokenIn/tokenOut is not the pool pair")
	ErrZeroAmountIn   = errors.New("daikon-prop: zero amountIn")
	ErrZeroAmountOut  = errors.New("daikon-prop: zero amountOut")
	ErrNotTrading     = errors.New("daikon-prop: market paused, inactive or never published")
	ErrExpired        = errors.New("daikon-prop: observation expired")
	ErrCapacity       = errors.New("daikon-prop: capacity exceeded")
	ErrVaultInventory = errors.New("daikon-prop: custody balance or allowance too low")
	ErrOverflow       = errors.New("daikon-prop: uint256 overflow")
)
