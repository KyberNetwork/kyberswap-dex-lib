package fermi

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeFermi

	defaultGas int64 = 163500

	fermiSwapperAddr     = "0xb1076fe3ab5e28005c7c323bac5ac06a680d452e"
	fermiEngineAddr      = "0x1038c87766e36d1925889e6f26d10e0012d50fed"
	fermiTraderVaultAddr = "0x585d44727129b9c69791b10238ca605932938b4f"

	methodGetPairs      = "getPairs"
	methodQuote         = "quoteAmounts"
	methodGetPairParams = "getPairParams"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amountIn: must be positive")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity: amountIn exceeds curve max")
	ErrZeroAmountOut         = errors.New("zero amountOut")
)
