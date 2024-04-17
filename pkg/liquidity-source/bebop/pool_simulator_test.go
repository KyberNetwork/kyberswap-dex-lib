package bebop

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

var entityPool = entity.Pool{
	Address:  "bebop_0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270_0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
	Exchange: "bebop",
	Type:     "bebop",
	Reserves: []string{"9320038994403940352", "166143156993"},
	Tokens: []*entity.PoolToken{
		{Address: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Decimals: 18, Swappable: true},
		{Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Decimals: 6, Swappable: true},
	},
	Extra: "{\"0to1\":[{\"q\":0.0001,\"p\":0.91245042136692},{\"q\":4.659919497201971,\"p\":0.91245042136692}," +
		"{\"q\":4.66001949720197,\"p\":0.91245042136692}]," +
		"\"1to0\":[{\"q\":0.0001,\"p\":1.0942398729806944},{\"q\":18277.528075741084,\"p\":1.0942398729806944}," +
		"{\"q\":25244.263002363805,\"p\":1.0939119116852096},{\"q\":32092.9359692824,\"p\":1.0937921053280593}," +
		"{\"q\":33219.273417201824,\"p\":1.0936723252106664},{\"q\":29917.17166407224,\"p\":1.0935525713244107}," +
		"{\"q\":27391.98476499627,\"p\":1.093432843660677}],\"tlrnce\":0}",
}

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.Equal(t, "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", poolSimulator.Token0.Address)
	assert.Equal(t, "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", poolSimulator.Token1.Address)
	assert.NotNil(t, poolSimulator.ZeroToOnePriceLevels)
	assert.NotNil(t, poolSimulator.OneToZeroPriceLevels)
	assert.Equal(t, []string{"0xc2132d05d31c914a87c6611c10748aeb04b58e8f"},
		poolSimulator.CanSwapTo("0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"))
	assert.Equal(t, []string{"0xc2132d05d31c914a87c6611c10748aeb04b58e8f"},
		poolSimulator.CanSwapFrom("0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"))
	assert.Equal(t, []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"},
		poolSimulator.CanSwapTo("0xc2132d05d31c914a87c6611c10748aeb04b58e8f"))
	assert.Equal(t, []string{"0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"},
		poolSimulator.CanSwapFrom("0xc2132d05d31c914a87c6611c10748aeb04b58e8f"))
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)

	tests := []struct {
		name              string
		amountIn          *big.Int
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:        "it should return error when swap higher than total level", // Total level ~166kMATIC
			amountIn:    big.NewInt(200_000_000000),
			expectedErr: ErrAmountInIsGreaterThanHighestPriceLevel,
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          big.NewInt(3_000_000),
			expectedAmountOut: bigIntFromString("3282719618942083072"),
		},
		{
			name:              "it should return correct amountOut when swap in all levels",
			amountIn:          big.NewInt(152_000_000),
			expectedAmountOut: bigIntFromString("166324460693065564160"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
					Amount: tc.amountIn,
				},
				TokenOut: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
			}

			result, err := poolSimulator.CalcAmountOut(params)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func bigIntFromString(s string) *big.Int {
	value, _ := new(big.Int).SetString(s, 10)
	return value
}
