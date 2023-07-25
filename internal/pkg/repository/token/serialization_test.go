package token

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestSerialization_EncodeToken(t *testing.T) {
	t.Run("it should encode token correctly", func(t *testing.T) {
		token := entity.Token{
			Address:     "0xABC",
			Symbol:      "ABC",
			Name:        "ABC Token",
			Decimals:    18,
			CgkID:       "abc",
			Type:        "erc20",
			PoolAddress: "xyz",
		}

		tokenStr, err := encodeToken(token)

		assert.NoError(t, err)
		assert.Equal(t, "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\",\"poolAddress\":\"xyz\"}", tokenStr)
	})
}

func TestSerialization_DecodeToken(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		member        string
		expectedToken *entity.Token
	}{
		{
			name:   "it should decode token correctly when it has full data",
			key:    "address",
			member: "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\",\"poolAddress\":\"xyz\"}",
			expectedToken: &entity.Token{
				Address:     "address",
				Symbol:      "ABC",
				Name:        "ABC Token",
				Decimals:    18,
				CgkID:       "abc",
				Type:        "erc20",
				PoolAddress: "xyz",
			},
		},
		{
			name:   "it should decode price correctly when it has no pool address data",
			key:    "address",
			member: "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\"}",
			expectedToken: &entity.Token{
				Address:     "address",
				Symbol:      "ABC",
				Name:        "ABC Token",
				Decimals:    18,
				CgkID:       "abc",
				Type:        "erc20",
				PoolAddress: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := decodeToken(test.key, test.member)

			assert.NoError(t, err)
			assert.Equal(t, test.expectedToken, token)
		})
	}
}
