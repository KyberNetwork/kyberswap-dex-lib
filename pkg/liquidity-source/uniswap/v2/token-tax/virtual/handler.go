package virtual

import tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"

func NewHandler(result tokentax.Result) tokentax.Handler {
	return tokentax.NewBasisPointHandler(result)
}
