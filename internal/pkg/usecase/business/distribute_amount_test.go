package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistributeAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		amount             *big.Int
		distributions      []uint64
		distributedAmounts []*big.Int
	}{
		{
			name:               "it should distribute amount correctly",
			amount:             big.NewInt(10_000_000_000),
			distributions:      []uint64{3000, 7000},
			distributedAmounts: []*big.Int{big.NewInt(3_000_000_000), big.NewInt(7_000_000_000)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			distributedAmounts := DistributeAmount(tc.amount, tc.distributions)

			assert.ElementsMatch(t, tc.distributedAmounts, distributedAmounts)
		})
	}
}

func TestCalcDistribution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		amount       *big.Int
		pathAmount   *big.Int
		distribution uint64
	}{
		{
			name:         "it should calculate distribution correctly",
			amount:       big.NewInt(10_000_000_000),
			pathAmount:   big.NewInt(3_000_000_000),
			distribution: 3000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			distribution := CalcDistribution(tc.amount, tc.pathAmount)

			assert.Equal(t, tc.distribution, distribution)
		})
	}
}
