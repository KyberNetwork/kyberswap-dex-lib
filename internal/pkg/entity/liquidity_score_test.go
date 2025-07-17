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
			name: "It should return encoded correctly with level 3 with pool has liquidiy score",
			poolScore: entity.PoolScore{
				LiquidityScore: 12345,
				Pool:           "0xabc",
				Level:          3,
				TvlInUsd:       100,
			},
			expectedScore: 31000000012345,
		},
		{
			name: "It should return encoded correctly with level 5 with pool has liquidiy score",
			poolScore: entity.PoolScore{
				LiquidityScore: 12345678,
				Pool:           "0xabc",
				Level:          5,
				TvlInUsd:       100,
			},
			expectedScore: 51000012345678,
		},
		{
			name: "It should return encoded correctly with maximum value of liquidity score",
			poolScore: entity.PoolScore{
				LiquidityScore: 999999999999,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       100,
			},
			expectedScore: 121999999999999,
		},
		{
			name: "It should return encoded correctly with tvl above upper bound",
			poolScore: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e13,
			},
			expectedScore: 120913242009132.4202,
		},
		{
			name: "It should return encoded correctly with tvl under upper bound",
			poolScore: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          2,
				TvlInUsd:       250,
			},
			expectedScore: 20000000000263.15789466759,
		},
		{
			name: "It should return encoded correctly with tvl around half upper bound",
			poolScore: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          6,
				TvlInUsd:       1e6,
			},

			expectedScore: 60000001052630.4709152938,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encodedScore := test.poolScore.EncodeScore()
			assert.Equal(t, test.expectedScore, encodedScore)
		})
	}
}

func TestPoolScore_CompareEncodeScore(t *testing.T) {
	type testInput struct {
		name       string
		poolScoreA entity.PoolScore
		poolScoreB entity.PoolScore
		ALessThanB bool
	}
	tests := []testInput{
		{
			name: "It should return compare between liquidity score correctly with the same level",
			poolScoreA: entity.PoolScore{
				LiquidityScore: 12345,
				Pool:           "0xabc",
				Level:          3,
				TvlInUsd:       100,
			},
			poolScoreB: entity.PoolScore{
				LiquidityScore: 123456,
				Pool:           "0xabc",
				Level:          3,
				TvlInUsd:       100,
			},
			ALessThanB: true,
		},
		{
			name: "It should return compare between liquidity score correctly with different level",
			poolScoreA: entity.PoolScore{
				LiquidityScore: 999999999999,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       100,
			},
			poolScoreB: entity.PoolScore{
				LiquidityScore: 999999999999,
				Pool:           "0xabc",
				Level:          9,
				TvlInUsd:       100,
			},
			ALessThanB: false,
		},
		{
			name: "It should return compare between tvl correctly with the same level",
			poolScoreA: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e13,
			},
			poolScoreB: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e14,
			},
			ALessThanB: true,
		},
		{
			name: "It should return compare between tvl correctly with different level",
			poolScoreA: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e13,
			},
			poolScoreB: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          6,
				TvlInUsd:       1e14,
			},
			ALessThanB: false,
		},
		{
			name: "It should return compare between tvl and liquidity score correctly with the same level",
			poolScoreA: entity.PoolScore{
				LiquidityScore: 999,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e13,
			},
			poolScoreB: entity.PoolScore{
				LiquidityScore: 0.0,
				Pool:           "0xabc",
				Level:          12,
				TvlInUsd:       1e15,
			},
			ALessThanB: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encodedScoreA := test.poolScoreA.EncodeScore()
			encodedScoreB := test.poolScoreB.EncodeScore()
			assert.Equal(t, encodedScoreA < encodedScoreB, test.ALessThanB)
		})
	}
}

func TestPoolScore_GetMinScore(t *testing.T) {
	type testInput struct {
		name             string
		amountInUsd      float64
		threshold        float64
		expectedMinScore float64
	}
	tests := []testInput{
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      4000,
			expectedMinScore: 10000000000000,
			threshold:        0,
		},
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      9999,
			expectedMinScore: 10000000000000,
			threshold:        0,
		},
		{
			name:             "It should return correct min score with least value less than amount in",
			amountInUsd:      10000,
			expectedMinScore: 20000000000000,
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
			expectedMinScore: 10000000000000,
			threshold:        3999,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			minScore := entity.GetMinScore(test.amountInUsd, test.threshold)
			assert.Equal(t, test.expectedMinScore, minScore)
		})
	}
}
