package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrice_Encode(t *testing.T) {
	t.Parallel()

	t.Run("it should encode correctly", func(t *testing.T) {
		price := Price{
			Price:             10000,
			Liquidity:         100000,
			LpAddress:         "lpAddress",
			MarketPrice:       120000,
			PreferPriceSource: PriceSourceKyberswap,
		}

		priceStr := price.Encode()

		assert.Equal(t, "{\"address\":\"\",\"price\":10000,\"liquidity\":100000,\"lpAddress\":\"lpAddress\",\"marketPrice\":120000,\"preferPriceSource\":\"kyberswap\"}", priceStr)
	})
}

func TestDecodePrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		key           string
		member        string
		expectedPrice Price
	}{
		{
			name:   "it should decode price correctly when it has full data",
			key:    "address",
			member: "{\"address\":\"address\",\"price\":10000,\"liquidity\":100000,\"lpAddress\":\"lpAddress\",\"marketPrice\":120000,\"preferPriceSource\":\"coingecko\"}",
			expectedPrice: Price{
				Address:           "address",
				Price:             10000,
				Liquidity:         100000,
				LpAddress:         "lpAddress",
				MarketPrice:       120000,
				PreferPriceSource: PriceSourceCoingecko,
			},
		},
		{
			name:   "it should decode price correctly when it has no market price data",
			key:    "address",
			member: "{\"address\":\"address\",\"price\":10000,\"liquidity\":100000,\"lpAddress\":\"lpAddress\",\"preferPriceSource\":\"kyberswap\"}",
			expectedPrice: Price{
				Address:           "address",
				Price:             10000,
				Liquidity:         100000,
				LpAddress:         "lpAddress",
				MarketPrice:       0,
				PreferPriceSource: PriceSourceKyberswap,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			price := DecodePrice(test.key, test.member)

			assert.Equal(t, test.expectedPrice, price)
		})
	}
}

func TestGetPreferredPrice(t *testing.T) {
	tests := []struct {
		name                  string
		key                   string
		price                 Price
		expectedPriceValue    float64
		expectedIsMarketPrice bool
	}{
		{
			name: "it should return price when the market price is 0 even if prefer price source is Coingecko",
			key:  "address",
			price: Price{
				Address:           "address",
				Price:             1,
				Liquidity:         100000,
				LpAddress:         "lpAddress",
				MarketPrice:       0,
				PreferPriceSource: PriceSourceCoingecko,
			},
			expectedPriceValue:    1,
			expectedIsMarketPrice: false,
		},
		{
			name: "it should return market price when the prefer price source is Coingecko",
			key:  "address",
			price: Price{
				Address:           "address",
				Price:             1,
				Liquidity:         100000,
				LpAddress:         "lpAddress",
				MarketPrice:       2,
				PreferPriceSource: PriceSourceCoingecko,
			},
			expectedPriceValue:    2,
			expectedIsMarketPrice: true,
		},
		{
			name: "it should return price when the prefer price source is Kyberswap",
			key:  "address",
			price: Price{
				Address:           "address",
				Price:             1,
				Liquidity:         100000,
				LpAddress:         "lpAddress",
				MarketPrice:       2,
				PreferPriceSource: PriceSourceKyberswap,
			},
			expectedPriceValue:    1,
			expectedIsMarketPrice: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			priceValue, isMarketPrice := test.price.GetPreferredPrice()
			assert.Equal(t, test.expectedPriceValue, priceValue)
			assert.Equal(t, test.expectedIsMarketPrice, isMarketPrice)
		})
	}
}
