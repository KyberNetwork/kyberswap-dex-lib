package kyberpmm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator_getAmountOut(t *testing.T) {
	type args struct {
		amountIn    *big.Float
		priceLevels []PriceLevel
	}
	tests := []struct {
		name              string
		args              args
		expectedAmountOut *big.Float
		expectedErr       error
	}{
		{
			name: "it should return error when price levels is empty",
			args: args{
				amountIn:    new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{},
			},
			expectedAmountOut: nil,
			expectedErr:       ErrEmptyPriceLevels,
		},
		{
			name: "it should return insufficient liquidity error when the requested amount is greater than available amount in price levels",
			args: args{
				amountIn: new(big.Float).SetFloat64(4),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       ErrInsufficientLiquidity,
		},
		{
			name: "it should return correct amount out when fully filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedAmountOut: new(big.Float).SetFloat64(100),
			expectedErr:       nil,
		},
		{
			name: "it should return correct amount out when partially filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedAmountOut: new(big.Float).SetFloat64(199),
			expectedErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amountOut, err := getAmountOut(tt.args.amountIn, tt.args.priceLevels)
			assert.Equal(t, tt.expectedErr, err)

			if amountOut != nil {
				assert.Equal(t, tt.expectedAmountOut.Cmp(amountOut), 0)
			}
		})
	}
}

func TestPoolSimulator_getNewPriceLevelsState(t *testing.T) {
	type args struct {
		amountIn    *big.Float
		priceLevels []PriceLevel
	}
	tests := []struct {
		name                string
		args                args
		expectedPriceLevels []PriceLevel
	}{
		{
			name: "it should do nothing when price levels is empty",
			args: args{
				amountIn:    new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when fully filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(1),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when the amountIn is greater than the amount available in the single price level",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when the amountIn is greater than the amount available in the all price levels",
			args: args{
				amountIn: new(big.Float).SetFloat64(5),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{},
		},
		{
			name: "it should return correct new price levels when partially filled",
			args: args{
				amountIn: new(big.Float).SetFloat64(2),
				priceLevels: []PriceLevel{
					{
						Price:  100,
						Amount: 1,
					},
					{
						Price:  99,
						Amount: 2,
					},
				},
			},
			expectedPriceLevels: []PriceLevel{
				{
					Price:  99,
					Amount: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newPriceLevels := getNewPriceLevelsState(tt.args.amountIn, tt.args.priceLevels)

			assert.ElementsMatch(t, tt.expectedPriceLevels, newPriceLevels)
		})
	}
}
