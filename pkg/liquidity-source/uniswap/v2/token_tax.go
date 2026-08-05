package uniswapv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func newTokenTaxTracker(
	chainID valueobject.ChainID, pool entity.Pool, extra Extra,
) (*tokentax.Tracker, tokentax.TaxInfo) {
	var info tokentax.TaxInfo
	if extra.TaxInfo != nil {
		info = *extra.TaxInfo
	}

	if !tokentax.SupportsChain(chainID) || len(pool.Tokens) != 2 {
		return nil, tokentax.TaxInfo{}
	}

	// A stored Checked=true doesn't guarantee this detector version has actually probed the token,
	// so force one recheck whenever the stored TaxCheckVersion is stale before trusting the cache.
	forceRecheck := info.TaxCheckVersion < currentTaxCheckVersion
	if !forceRecheck && info.Checked && info.Token == "" {
		return nil, info
	}

	return tokentax.NewTracker(chainID, pool.Tokens[0].Address, pool.Tokens[1].Address, info), tokentax.TaxInfo{}
}
