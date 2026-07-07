package nadfun

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGetFeeAmount(t *testing.T) {
	tests := []struct {
		name        string
		amount      *uint256.Int
		protocolFee *uint256.Int
		expectedFee *uint256.Int
	}{
		{
			name:        "1% fee on 1,000,000",
			amount:      uint256.NewInt(1000000),
			protocolFee: uint256.NewInt(10000), // 1%
			expectedFee: uint256.NewInt(10000),
		},
		{
			name:        "1% fee on 100,000,000",
			amount:      uint256.NewInt(100000000),
			protocolFee: uint256.NewInt(10000), // 1%
			expectedFee: uint256.NewInt(1000000),
		},
		{
			name:        "1% fee on 10,000,000,000",
			amount:      uint256.NewInt(10000000000),
			protocolFee: uint256.NewInt(10000), // 1%
			expectedFee: uint256.NewInt(100000000),
		},
		{
			name:        "0% fee",
			amount:      uint256.NewInt(1000000),
			protocolFee: uint256.NewInt(0),
			expectedFee: uint256.NewInt(0),
		},
		{
			name:        "0.5% fee on 1,000,000",
			amount:      uint256.NewInt(1000000),
			protocolFee: uint256.NewInt(5000), // 0.5%
			expectedFee: uint256.NewInt(5000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := getFeeAmount(tt.amount, tt.protocolFee)
			require.Equal(t, tt.expectedFee, fee, "Fee calculation mismatch")
		})
	}
}
