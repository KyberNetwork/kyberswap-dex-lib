package synthetix

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/stretchr/testify/assert"
)

func TestValidator_CheckAtomicVolume(t *testing.T) {
	testCases := []struct {
		name                    string
		sourceSusdValue         *big.Int
		blockTimestamp          uint64
		atomicMaxVolumePerBlock *big.Int
		lastAtomicVolume        *synthetix.ExchangeVolumeAtPeriod
		expectedErr             error
	}{
		{
			name:                    "it should return error when lastAtomicVolume is invalid",
			sourceSusdValue:         big.NewInt(100000),
			blockTimestamp:          100,
			atomicMaxVolumePerBlock: big.NewInt(1000000),
			lastAtomicVolume:        nil,
			expectedErr:             synthetix.ErrInvalidLastAtomicVolume,
		},
		{
			name:                    "it should return error when volume limit is surpassed",
			sourceSusdValue:         big.NewInt(1100000),
			blockTimestamp:          100,
			atomicMaxVolumePerBlock: big.NewInt(1000000),
			lastAtomicVolume: &synthetix.ExchangeVolumeAtPeriod{
				Time:   200,
				Volume: big.NewInt(200000),
			},
			expectedErr: synthetix.ErrSurpassedVolumeLimit,
		},
		{
			name:                    "it should return nil when volume per block is within limit and current block timestamp is equal lastAtomicVolume time",
			sourceSusdValue:         big.NewInt(100000),
			blockTimestamp:          100,
			atomicMaxVolumePerBlock: big.NewInt(1000000),
			lastAtomicVolume: &synthetix.ExchangeVolumeAtPeriod{
				Time:   100,
				Volume: big.NewInt(200000),
			},
			expectedErr: nil,
		},
		{
			name:                    "it should return nil when volume per block is within limit",
			sourceSusdValue:         big.NewInt(100000),
			blockTimestamp:          100,
			atomicMaxVolumePerBlock: big.NewInt(1000000),
			lastAtomicVolume: &synthetix.ExchangeVolumeAtPeriod{
				Time:   200,
				Volume: big.NewInt(200000),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkAtomicVolume(tc.sourceSusdValue, tc.blockTimestamp, tc.atomicMaxVolumePerBlock, tc.lastAtomicVolume)

			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
