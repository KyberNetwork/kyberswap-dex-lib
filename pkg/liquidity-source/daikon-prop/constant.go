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

	// MaxFeePpm is the 5% cap the pair applies to swapData fees; larger values are cut.
	MaxFeePpm = 50_000

	// Hop gas measured with Tenderly through executeGeneric on Base (test-dex): identical across
	// amounts and runs, so each is its own average.
	gasBuyBase  int64 = 109_759 // quote in
	gasSellBase int64 = 115_276 // base in
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
