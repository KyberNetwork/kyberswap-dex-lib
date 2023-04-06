package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken_Encode(t *testing.T) {
	t.Parallel()

	t.Run("it should encode token correctly", func(t *testing.T) {
		token := Token{
			Address:     "0xABC",
			Symbol:      "ABC",
			Name:        "ABC Token",
			Decimals:    18,
			CgkID:       "abc",
			Type:        "erc20",
			PoolAddress: "xyz",
		}

		tokenStr := token.Encode()

		assert.Equal(t, "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\",\"poolAddress\":\"xyz\"}", tokenStr)
	})
}

func TestDecodeToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		key           string
		member        string
		expectedToken Token
	}{
		{
			name:   "it should decode token correctly when it has full data",
			key:    "address",
			member: "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\",\"poolAddress\":\"xyz\"}",
			expectedToken: Token{
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
			expectedToken: Token{
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
			token := DecodeToken(test.key, test.member)

			assert.Equal(t, test.expectedToken, token)
		})
	}
}
