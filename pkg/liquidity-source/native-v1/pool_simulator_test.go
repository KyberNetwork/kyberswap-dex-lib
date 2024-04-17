package nativev1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var entityPool = entity.Pool{
	Address:  "native_v1_0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270_0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
	Exchange: "native-v1",
	Type:     "native-v1",
	Reserves: []string{"9320038994403940352", "166143156993"},
	Tokens: []*entity.PoolToken{
		{Address: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Decimals: 18, Swappable: true},
		{Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Decimals: 6, Swappable: true},
	},
	Extra: "{\"0to1\":[{\"q\":0.0001,\"p\":0.91245042136692},{\"q\":4.659919497201971,\"p\":0.91245042136692}," +
		"{\"q\":4.66001949720197,\"p\":0.90924546691228}],\"min0\":0.0001," +
		"\"1to0\":[{\"q\":0.0001,\"p\":1.0942398729806944},{\"q\":18277.528075741084,\"p\":1.0942398729806944}," +
		"{\"q\":25244.263002363805,\"p\":1.0939119116852096},{\"q\":32092.9359692824,\"p\":1.0937921053280593}," +
		"{\"q\":33219.273417201824,\"p\":1.0936723252106664},{\"q\":29917.17166407224,\"p\":1.0935525713244107}," +
		"{\"q\":27391.98476499627,\"p\":1.093432843660677}],\"min1\":0.0001,\"tlrnce\":0}",
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
		name                 string
		amountIn0, amountIn1 *big.Int
		expectedAmountOut    *big.Int
		expectedErr          error
	}{
		{
			name:        "it should return error when swap lower than min0 level", // Lowest level 0.0001MATIC
			amountIn0:   big.NewInt(1_000000000),
			expectedErr: ErrAmountInIsLessThanLowestPriceLevel,
		},
		{
			name:        "it should return error when swap lower than min1 level", // Lowest level 0.0001USDT
			amountIn1:   big.NewInt(99),
			expectedErr: ErrAmountInIsLessThanLowestPriceLevel,
		},
		{
			name:        "it should return error when swap higher than total level", // Total level ~166kUSDT
			amountIn1:   big.NewInt(200_000_000000),
			expectedErr: ErrAmountInIsGreaterThanHighestPriceLevel,
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn1:         big.NewInt(3_000_000),
			expectedAmountOut: bigIntFromString("3282719618942082560"),
		},
		{
			name:              "it should return correct amountOut when swap from token0",
			amountIn0:         big.NewInt(6_000000000_000000000),
			expectedAmountOut: bigIntFromString("5470407"),
		},
		{
			name:              "it should return correct amountOut when swap in all levels",
			amountIn1:         big.NewInt(152_000_000),
			expectedAmountOut: bigIntFromString("166324460693065564160"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenIn, tokenOut, amountIn := entityPool.Tokens[0].Address, entityPool.Tokens[1].Address, tt.amountIn0
			if amountIn == nil {
				tokenIn, tokenOut, amountIn = tokenOut, tokenIn, tt.amountIn1
			}
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
			}

			result, err := poolSimulator.CalcAmountOut(params)
			assert.Equal(t, tt.expectedErr, err)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func bigIntFromString(s string) *big.Int {
	value, _ := new(big.Int).SetString(s, 10)
	return value
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	tests := []struct {
		name                         string
		amountIn0, amountIn1         *big.Int
		expectedZeroToOnePriceLevels []PriceLevel
		expectedOneToZeroPriceLevels []PriceLevel
	}{
		{
			name:      "min token0",
			amountIn0: big.NewInt(100_000_000000000),
			expectedZeroToOnePriceLevels: []PriceLevel{
				{Quote: 4.659919497201971, Price: 0.91245042136692},
				{Quote: 4.66001949720197, Price: 0.90924546691228},
			},
			expectedOneToZeroPriceLevels: []PriceLevel{
				{Quote: 0.0001, Price: 1.0942398729806944},
				{Quote: 18277.528075741084, Price: 1.0942398729806944},
				{Quote: 25244.263002363805, Price: 1.0939119116852096},
				{Quote: 32092.9359692824, Price: 1.0937921053280593},
				{Quote: 33219.273417201824, Price: 1.0936723252106664},
				{Quote: 29917.17166407224, Price: 1.0935525713244107},
				{Quote: 27391.98476499627, Price: 1.093432843660677}},
		},
		{
			name:      "token0",
			amountIn0: big.NewInt(5_000000000_000000000),
			expectedZeroToOnePriceLevels: []PriceLevel{
				{Quote: 4.320038994403941, Price: 0.90924546691228},
			},
			expectedOneToZeroPriceLevels: []PriceLevel{
				{Quote: 0.0001, Price: 1.0942398729806944},
				{Quote: 18277.528075741084, Price: 1.0942398729806944},
				{Quote: 25244.263002363805, Price: 1.0939119116852096},
				{Quote: 32092.9359692824, Price: 1.0937921053280593},
				{Quote: 33219.273417201824, Price: 1.0936723252106664},
				{Quote: 29917.17166407224, Price: 1.0935525713244107},
				{Quote: 27391.98476499627, Price: 1.093432843660677}},
		},
		{
			name:      "token1",
			amountIn1: big.NewInt(10000_000000),
			expectedZeroToOnePriceLevels: []PriceLevel{
				{Quote: 0.0001, Price: 0.91245042136692},
				{Quote: 4.659919497201971, Price: 0.91245042136692},
				{Quote: 4.66001949720197, Price: 0.90924546691228}},
			expectedOneToZeroPriceLevels: []PriceLevel{
				{Quote: 8277.528175741083, Price: 1.0942398729806944},
				{Quote: 25244.263002363805, Price: 1.0939119116852096},
				{Quote: 32092.9359692824, Price: 1.0937921053280593},
				{Quote: 33219.273417201824, Price: 1.0936723252106664},
				{Quote: 29917.17166407224, Price: 1.0935525713244107},
				{Quote: 27391.98476499627, Price: 1.093432843660677}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPoolSimulator(entityPool)
			assert.NoError(t, err)
			token, amountIn := entityPool.Tokens[0].Address, tt.amountIn0
			if amountIn == nil {
				token, amountIn = entityPool.Tokens[1].Address, tt.amountIn1
			}
			p.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: token, Amount: amountIn}})
			assert.Equal(t, tt.expectedZeroToOnePriceLevels, p.ZeroToOnePriceLevels)
			assert.Equal(t, tt.expectedOneToZeroPriceLevels, p.OneToZeroPriceLevels)
		})
	}
}
