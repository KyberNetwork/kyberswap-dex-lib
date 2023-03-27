package dto

import "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"

type (
	GetTokensResult struct {
		Tokens []*GetTokensResultToken `json:"tokens"`
	}

	GetTokensResultToken struct {
		Name      string                    `json:"name"`
		Decimals  uint8                     `json:"decimals"`
		Symbol    string                    `json:"symbol"`
		Type      string                    `json:"type"`
		Price     float64                   `json:"price"`
		Liquidity float64                   `json:"liquidity,omitempty"`
		LPAddress string                    `json:"lpAddress,omitempty"`
		Pool      *GetTokensResultTokenPool `json:"pool,omitempty"`
	}

	GetTokensResultTokenPool struct {
		Address     string                           `json:"address"`
		TotalSupply float64                          `json:"totalSupply"`
		ReserveUSD  float64                          `json:"reserveUsd"`
		Tokens      []*GetTokensResultTokenPoolToken `json:"tokens"`
	}

	GetTokensResultTokenPoolToken struct {
		Name     string `json:"name"`
		Decimals uint8  `json:"decimals"`
		Symbol   string `json:"symbol"`
		Type     string `json:"type"`
		Weight   uint   `json:"weight"`
	}
)

type GetTokensResultTokenBuilder struct {
	showExtra      bool
	showPoolTokens bool

	token *GetTokensResultToken
}

func NewGetTokensResultTokenBuilder(
	showExtra bool,
	showPoolTokens bool,
) *GetTokensResultTokenBuilder {
	return &GetTokensResultTokenBuilder{
		showExtra:      showExtra,
		showPoolTokens: showPoolTokens,

		token: &GetTokensResultToken{},
	}
}

func (b *GetTokensResultTokenBuilder) WithToken(token entity.Token) IGetTokensResultTokenBuilder {
	b.token.Name = token.Name
	b.token.Decimals = token.Decimals
	b.token.Symbol = token.Symbol
	b.token.Type = token.Type

	return b
}

func (b *GetTokensResultTokenBuilder) WithPrice(price entity.Price) IGetTokensResultTokenBuilder {
	b.token.Price = price.Price

	if b.showExtra {
		b.token.Liquidity = price.Liquidity
		b.token.LPAddress = price.LpAddress
	}

	return b
}

func (b *GetTokensResultTokenBuilder) WithPool(
	pool entity.Pool,
	tokenByAddress map[string]entity.Token,
) IGetTokensResultTokenBuilder {
	if !b.showPoolTokens {
		return b
	}

	if pool.IsZero() {
		return b
	}

	resultPoolToken := make([]*GetTokensResultTokenPoolToken, 0, len(pool.Tokens))
	for _, pToken := range pool.Tokens {
		token, ok := tokenByAddress[pToken.Address]
		if !ok {
			resultPoolToken = append(
				resultPoolToken,
				&GetTokensResultTokenPoolToken{
					Weight: pToken.Weight,
				},
			)
			continue
		}

		resultPoolToken = append(
			resultPoolToken,
			&GetTokensResultTokenPoolToken{
				Name:     token.Name,
				Decimals: token.Decimals,
				Symbol:   token.Symbol,
				Type:     token.Type,
				Weight:   pToken.Weight,
			})
	}

	b.token.Pool = &GetTokensResultTokenPool{
		Address:     pool.Address,
		TotalSupply: pool.GetTotalSupply(),
		ReserveUSD:  pool.ReserveUsd,
		Tokens:      resultPoolToken,
	}

	return b
}

func (b *GetTokensResultTokenBuilder) GetToken() *GetTokensResultToken {
	return b.token
}
