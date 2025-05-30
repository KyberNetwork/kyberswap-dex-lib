package valueobject

import (
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	TokenGroupTypeConverter      = "Converter"
	TokenGroupTypeCorrelated     = "Correlated"
	TokenGroupTypeDefault        = "Default"
	TokenGroupTypeStable         = "Stable"
	TokenGroupTypeStrictlyStable = "StrictlyStable"
)

type TokenGroupParams struct {
	TokenIn  string
	TokenOut string
	Exchange string
}

func (g *TokenGroupConfig) GetTokenGroupType(params TokenGroupParams) (string, bool) {
	// Check converter exchanges
	switch params.Exchange {
	case dexValueObject.ExchangeFrxETH, dexValueObject.ExchangeDaiUsds,
		dexValueObject.ExchangeUsd0PP, dexValueObject.ExchangeOETH,
		dexValueObject.ExchangePolMatic, dexValueObject.ExchangeEtherFieBTC,
		dexValueObject.ExchangeHoney:
		return TokenGroupTypeConverter, true
	}

	// Check type by pool types
	if dexValueObject.IsRFQSource(dexValueObject.Exchange(params.Exchange)) {
		return TokenGroupTypeStrictlyStable, true
	}

	// Check type by tokens
	// Reference: https://www.notion.so/kybernetwork/Stable-and-Correlated-Tokens-data-d1bdc7ad1ec14d8ebeab031c493e730e
	if g.StableGroup[params.TokenIn] && g.StableGroup[params.TokenOut] {
		return TokenGroupTypeStable, true
	} else if g.CorrelatedGroup1[params.TokenIn] && g.CorrelatedGroup1[params.TokenOut] {
		return TokenGroupTypeCorrelated, true
	} else if g.CorrelatedGroup2[params.TokenIn] && g.CorrelatedGroup2[params.TokenOut] {
		return TokenGroupTypeCorrelated, true
	} else if g.CorrelatedGroup3[params.TokenIn] && g.CorrelatedGroup3[params.TokenOut] {
		return TokenGroupTypeCorrelated, true
	}

	return TokenGroupTypeDefault, false
}
