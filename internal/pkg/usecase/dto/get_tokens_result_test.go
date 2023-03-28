package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func TestNewGetTokensResultTokenBuilder(t *testing.T) {
	t.Run("it should return correct builder", func(t *testing.T) {
		expected := &GetTokensResultTokenBuilder{
			showExtra:      true,
			showPoolTokens: false,

			token: &GetTokensResultToken{},
		}

		assert.Equal(t, expected, NewGetTokensResultTokenBuilder(true, false))
	})
}

func TestGetTokensResultTokenBuilder_WithToken(t *testing.T) {
	t.Run("it should build correct token data", func(t *testing.T) {
		token := entity.Token{
			Name:     "name",
			Decimals: 6,
			Symbol:   "symbol",
			Type:     "type",
		}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      true,
			showPoolTokens: true,

			token: &GetTokensResultToken{
				Name:     "name",
				Decimals: 6,
				Symbol:   "symbol",
				Type:     "type",
			},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      true,
			showPoolTokens: true,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithToken(token))
	})
}

func TestGetTokensResultTokenBuilder_WithPrice(t *testing.T) {
	t.Run("it should build correct price data when showExtra is false", func(t *testing.T) {
		price := entity.Price{
			Price:     float64(1000000),
			Liquidity: float64(20000),
			LpAddress: "poolAddress",
		}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: true,

			token: &GetTokensResultToken{
				Price: float64(1000000),
			},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: true,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithPrice(price))
	})

	t.Run("it should build correct price data when showExtra is true", func(t *testing.T) {
		price := entity.Price{
			Price:     float64(1000000),
			Liquidity: float64(20000),
			LpAddress: "poolAddress",
		}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      true,
			showPoolTokens: true,

			token: &GetTokensResultToken{
				Price:     float64(1000000),
				Liquidity: float64(20000),
				LPAddress: "poolAddress",
			},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      true,
			showPoolTokens: true,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithPrice(price))
	})
}

func TestGetTokensResultTokenBuilder_WithPool(t *testing.T) {
	t.Run("it should skip when showPoolTokens is false", func(t *testing.T) {
		pool := entity.Pool{
			Address:     "poolAddress",
			TotalSupply: "1000000",
			ReserveUsd:  1000000,
			Tokens: []*entity.PoolToken{
				{
					Address: "token0Address",
					Weight:  50,
				},
				{
					Address: "token1Address",
					Weight:  50,
				},
			},
		}
		tokenByAddress := map[string]entity.Token{
			"token0Address": {
				Name:     "token0Name",
				Decimals: 6,
				Symbol:   "token0",
				Type:     "type0",
			},
			"token1Address": {
				Name:     "token1Name",
				Decimals: 18,
				Symbol:   "token1",
				Type:     "type1",
			},
		}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: false,

			token: &GetTokensResultToken{},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: false,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithPool(pool, tokenByAddress))
	})

	t.Run("it should skip when pool is zero", func(t *testing.T) {
		pool := entity.Pool{}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: false,

			token: &GetTokensResultToken{},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: false,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithPool(pool, nil))
	})

	t.Run("it should return correct pool when showPoolTokens is true and pool is not zero", func(t *testing.T) {
		pool := entity.Pool{
			Address:     "poolAddress",
			TotalSupply: "1000000000000000000000000",
			ReserveUsd:  1000000,
			Tokens: []*entity.PoolToken{
				{
					Address: "token0Address",
					Weight:  50,
				},
				{
					Address: "token1Address",
					Weight:  50,
				},
			},
		}
		tokenByAddress := map[string]entity.Token{
			"token0Address": {
				Name:     "token0Name",
				Decimals: 6,
				Symbol:   "token0",
				Type:     "type0",
			},
			"token1Address": {
				Name:     "token1Name",
				Decimals: 18,
				Symbol:   "token1",
				Type:     "type1",
			},
		}

		expected := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: true,

			token: &GetTokensResultToken{
				Pool: &GetTokensResultTokenPool{
					Address:     "poolAddress",
					TotalSupply: 1000000,
					ReserveUSD:  1000000,
					Tokens: []*GetTokensResultTokenPoolToken{
						{
							Name:     "token0Name",
							Decimals: 6,
							Symbol:   "token0",
							Type:     "type0",
							Weight:   50,
						},
						{
							Name:     "token1Name",
							Decimals: 18,
							Symbol:   "token1",
							Type:     "type1",
							Weight:   50,
						},
					},
				},
			},
		}

		builder := &GetTokensResultTokenBuilder{
			showExtra:      false,
			showPoolTokens: true,
			token:          &GetTokensResultToken{},
		}

		assert.Equal(t, expected, builder.WithPool(pool, tokenByAddress))
	})
}

func TestGetTokensResultTokenBuilder_GetToken(t *testing.T) {
	t.Run("it should return correct token", func(t *testing.T) {
		builder := &GetTokensResultTokenBuilder{
			token: &GetTokensResultToken{
				Name: "tokenA",
			},
		}

		assert.Equal(t, &GetTokensResultToken{Name: "tokenA"}, builder.GetToken())
	})
}
