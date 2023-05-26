package price

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func TestPriceRedisRepository_EncodePrice(t *testing.T) {
	t.Run("it should encode price correctly", func(t *testing.T) {
		price := entity.Price{
			Address:           "0x111",
			Price:             1000,
			Liquidity:         1000,
			LpAddress:         "0xLP",
			MarketPrice:       1001,
			PreferPriceSource: "kyberswap",
		}

		priceStr, err := encodePrice(price)

		assert.NoError(t, err)
		assert.Equal(t, "{\"address\":\"0x111\",\"price\":1000,\"liquidity\":1000,\"lpAddress\":\"0xLP\",\"marketPrice\":1001,\"preferPriceSource\":\"kyberswap\"}", priceStr)
	})
}

func TestPriceRedisRepository_DecodePrice(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		member        string
		expectedPrice *entity.Price
	}{
		{
			name:   "it should decode price correctly when it has full data",
			key:    "0x111",
			member: "{\"address\":\"0x111\",\"price\":1000,\"liquidity\":1000,\"lpAddress\":\"0xLP1\",\"marketPrice\":1001,\"preferPriceSource\":\"kyberswap\"}",
			expectedPrice: &entity.Price{
				Address:           "0x111",
				Price:             1000,
				Liquidity:         1000,
				LpAddress:         "0xLP1",
				MarketPrice:       1001,
				PreferPriceSource: "kyberswap",
			},
		},
		{
			name:   "it should decode price correctly when it has no PreferPriceSource data",
			key:    "0x222",
			member: "{\"address\":\"0x222\",\"price\":2000,\"liquidity\":2000,\"lpAddress\":\"0xLP2\",\"marketPrice\":2002}",
			expectedPrice: &entity.Price{
				Address:     "0x222",
				Price:       2000,
				Liquidity:   2000,
				LpAddress:   "0xLP2",
				MarketPrice: 2002,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			price, err := decodePrice(test.key, test.member)

			assert.NoError(t, err)
			assert.Equal(t, test.expectedPrice, price)
		})
	}
}
