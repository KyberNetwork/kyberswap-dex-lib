package token_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
)

func TestSerialization_DecodeFullToken(t *testing.T) {
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
				Address:     "0xABC",
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
				Address:     "0xABC",
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
			token, err := token.DecodeToken[entity.Token](context.TODO(), test.member, test.key)

			assert.NoError(t, err)
			assert.Equal(t, test.expectedToken, token)
		})
	}
}

func TestSerialization_DecodeToken(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		member        string
		expectedToken *entity.SimplifiedToken
	}{
		{
			name:   "it should decode token correctly when it has full data",
			key:    "address",
			member: "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\",\"poolAddress\":\"xyz\"}",
			expectedToken: &entity.SimplifiedToken{
				Address:  "0xABC",
				Decimals: 18,
			},
		},
		{
			name:   "it should decode price correctly when it has no pool address data",
			key:    "address",
			member: "{\"address\":\"0xABC\",\"symbol\":\"ABC\",\"name\":\"ABC Token\",\"decimals\":18,\"cgkId\":\"abc\",\"type\":\"erc20\"}",
			expectedToken: &entity.SimplifiedToken{
				Address:  "0xABC",
				Decimals: 18,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := token.DecodeToken[entity.SimplifiedToken](context.TODO(), test.member, test.key)

			assert.NoError(t, err)
			assert.Equal(t, test.expectedToken, token)
		})
	}
}
