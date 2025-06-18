package valueobject

const (
	TokenGroupTypeCorrelated = "Correlated"
	TokenGroupTypeDefault    = "Default"
	TokenGroupTypeStable     = "Stable"
)

type TokenGroupParams struct {
	TokenIn  string
	TokenOut string
	Exchange string
}

func (g *TokenGroupConfig) GetTokenGroupType(params TokenGroupParams) (string, bool) {
	// Check type by tokens
	// Reference: https://www.notion.so/kybernetwork/Stable-and-Correlated-Tokens-data-d1bdc7ad1ec14d8ebeab031c493e730e
	if g == nil {
		return TokenGroupTypeDefault, false
	} else if g.StableGroup[params.TokenIn] && g.StableGroup[params.TokenOut] {
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
