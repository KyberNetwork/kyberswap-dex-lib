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

		assert.Equal(t, "ABC:ABC Token:18:abc:erc20:xyz", tokenStr)
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
			member: "ABC:ABC Token:18:abc:erc20:xyz",
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
			member: "ABC:ABC Token:18:abc:erc20",
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
