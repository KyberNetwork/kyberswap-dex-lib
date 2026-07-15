package uniswapv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	fourmeme "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/four-meme"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/virtual"
)

func newTokenTaxTracker(factory string, pool entity.Pool, extra Extra) (tokentax.Tracker, tokentax.TaxInfo) {
	var info tokentax.TaxInfo
	if extra.TaxInfo != nil {
		info = *extra.TaxInfo
	}

	switch {
	case virtual.SupportsFactory(factory):
		tokenAddress := virtual.FindTaxToken(pool)
		if tokenAddress == "" || info.Checked && info.Token == "" {
			info.Checked = true
			return nil, info
		}
		return virtual.NewTracker(pool.Address, tokenAddress, factory, info), tokentax.TaxInfo{}
	case fourmeme.SupportsFactory(factory):
		tokenAddress := fourmeme.FindTaxToken(pool)
		if tokenAddress == "" || info.Checked && info.Token == "" {
			info.Checked = true
			return nil, info
		}
		return fourmeme.NewTracker(pool.Address, tokenAddress, info), tokentax.TaxInfo{}
	default:
		return nil, tokentax.TaxInfo{}
	}
}

func newTokenTaxHandler(info *tokentax.TaxInfo) tokentax.Handler {
	if info == nil {
		return tokentax.Handler{}
	}
	switch info.Protocol {
	case virtual.Protocol, fourmeme.Protocol:
		return tokentax.NewHandler(*info)
	default:
		return tokentax.Handler{}
	}
}
