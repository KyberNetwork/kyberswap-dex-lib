package entity_test

import (
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func TestPoolScore_EncodeScore(t *testing.T) {
	type testInput struct {
		name          string
		poolScore     entity.PoolScore
		expectedScore float64
	}
	tests := []testInput{
		{
			name: "It should return encoded correctly with level 3",
			poolScore: entity.PoolScore{
				LiquidityScore: 12345,
				Pool:           "0xabc",
				Level:          3,
			},
			expectedScore: 3000000012345,
		},
		{
			name: "It should return encoded correctly with level 5",
			poolScore: entity.PoolScore{
				LiquidityScore: 12345678,
				Pool:           "0xabc",
				Level:          5,
			},
			expectedScore: 5000012345678,
		},
		{
			name: "It should return encoded correctly with maximum value of liquidity score",
			poolScore: entity.PoolScore{
				LiquidityScore: 999999999999,
				Pool:           "0xabc",
				Level:          12,
			},
			expectedScore: 12999999999999,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encodedScore := test.poolScore.EncodeScore(true)
			assert.Equal(t, test.expectedScore, encodedScore)
		})
	}
}

func TestPoolScore_GetMinScore(t *testing.T) {
	type testInput struct {
		name             string
		amountInUsd      float64
		threshold        float64
		expectedMinScore float64
		err              error
	}
	tests := []testInput{
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      4000,
			expectedMinScore: 3000000000000,
			threshold:        0,
		},
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      9999,
			expectedMinScore: 3000000000000,
			threshold:        0,
		},
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      10000,
			expectedMinScore: 4000000000000,
			threshold:        9000,
		},
		{
			name:             "It should return correct min score 0 when amount in less than threshold",
			amountInUsd:      3999,
			expectedMinScore: 0,
			threshold:        4000,
		},
		{
			name:             "It should return correct min score when amount in greater than threshold",
			amountInUsd:      4000,
			expectedMinScore: 3000000000000,
			threshold:        3999,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			minScore, err := entity.GetMinScore(test.amountInUsd, test.threshold)
			if test.err != nil {
				assert.Equal(t, test.err.Error(), err.Error())
			}
			assert.Equal(t, test.expectedMinScore, minScore)
		})
	}
}
