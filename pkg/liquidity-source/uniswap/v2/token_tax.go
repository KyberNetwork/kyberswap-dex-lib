package uniswapv2

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	fourmeme "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/four-meme"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/virtual"
)

func newTokenTaxTracker(factory string, pool entity.Pool, previous tokentax.Result) tokentax.Tracker {
	if previous.Checked && previous.TokenAddress == "" {
		return tokentax.NewStaticTracker(previous)
	}

	switch {
	case virtual.SupportsFactory(factory):
		return virtual.NewTracker(pool)
	case fourmeme.SupportsFactory(factory):
		return fourmeme.NewTracker(pool, previous)
	default:
		return tokentax.NewStaticTracker(tokentax.Result{Checked: true})
	}
}

func newTokenTaxHandler(result tokentax.Result) tokentax.Handler {
	switch result.Protocol {
	case virtual.Protocol:
		return virtual.NewHandler(result)
	case fourmeme.Protocol:
		return fourmeme.NewHandler(result)
	default:
		return tokentax.NoopHandler{}
	}
}

func tokenTaxResult(pool entity.Pool, extra Extra) tokentax.Result {
	return tokentax.Result{
		Protocol:     extra.TaxProtocol,
		TokenAddress: tokenAtIndex(pool, extra.TaxTokenIndex),
		BuyTaxBps:    extra.BuyTaxBps,
		SellTaxBps:   extra.SellTaxBps,
		Checked:      extra.TaxChecked,
	}
}

func tokenAtIndex(pool entity.Pool, index int) string {
	if index < 0 || index >= len(pool.Tokens) {
		return ""
	}
	return strings.ToLower(pool.Tokens[index].Address)
}

func findTokenIndex(tokens []*entity.PoolToken, tokenAddress string) int {
	for i, token := range tokens {
		if strings.EqualFold(token.Address, tokenAddress) {
			return i
		}
	}
	return -1
}
