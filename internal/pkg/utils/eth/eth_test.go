package eth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestIsEther(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		token  string
		result bool
	}{
		{
			name:   "it should be true when token is equal ether",
			token:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			result: true,
		},
		{
			name:   "it should be true when token is equal fold to ether",
			token:  "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			result: true,
		},
		{
			name:   "it should be false when token is not equal to ether",
			token:  "abc",
			result: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsEther(tc.token)

			assert.Equal(t, tc.result, result)
		})
	}
}

func TestIsWETH(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		token   string
		chainID valueobject.ChainID
		result  bool
	}{
		{
			name:    "it should be true when token is equal to WETH",
			token:   "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			chainID: valueobject.ChainIDEthereum,
			result:  true,
		},
		{
			name:    "it should be true when token is equal fold to WETH",
			token:   "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			chainID: valueobject.ChainIDEthereum,
			result:  true,
		},
		{
			name:    "it should be false when token is not equal to ether",
			token:   "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			chainID: valueobject.ChainIDEthereum,
			result:  false,
		},
		{
			name:    "it should return false when weth not found",
			token:   "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			chainID: valueobject.ChainID(0),
			result:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWETH(tc.token, tc.chainID)

			assert.Equal(t, tc.result, result)
		})
	}
}

func TestConvertEtherToWETH(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		token         string
		chainID       valueobject.ChainID
		expectedToken string
		expectedErr   error
	}{
		{
			name:          "it should return token when token is not ether",
			token:         "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			chainID:       valueobject.ChainIDEthereum,
			expectedToken: "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			expectedErr:   nil,
		},
		{
			name:          "it should return token with error when weth not found",
			token:         "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			chainID:       valueobject.ChainID(0),
			expectedToken: "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			expectedErr:   ErrWETHNotFound,
		},
		{
			name:          "it should return weth when token is ether",
			token:         "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			chainID:       valueobject.ChainIDEthereum,
			expectedToken: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expectedErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertEtherToWETH(tc.token, tc.chainID)

			assert.Equal(t, tc.expectedToken, result)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestConvertWETHToEther(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		token         string
		chainID       valueobject.ChainID
		expectedToken string
		expectedErr   error
	}{
		{
			name:          "it should return token when weth not found",
			token:         "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			chainID:       valueobject.ChainID(0),
			expectedToken: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			expectedErr:   ErrWETHNotFound,
		},
		{
			name:          "it should return ether address when token is weth",
			token:         "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			chainID:       valueobject.ChainIDEthereum,
			expectedToken: "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			expectedErr:   nil,
		}, {
			name:          "it should return token when token is not weth",
			token:         "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			chainID:       valueobject.ChainIDEthereum,
			expectedToken: "0xc778417E063141139Fce010982780140Aa0cD5Ab",
			expectedErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertWETHToEther(tc.token, tc.chainID)

			assert.Equal(t, tc.expectedToken, result)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
