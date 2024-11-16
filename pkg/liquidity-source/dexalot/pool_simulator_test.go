package dexalot

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

/*
0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab // eth
0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e // usdc
{
	"prices": {
		"WETH/USDC": {
			"bids": [ // descending: price of base | amount of quote
					["100", "200"], 2 * 100 usdc
					["80", "320"],  4 * 80 usdc
					["60", "600"],  6 * 60 usdc
			],
			"asks": [ // ascending: price of base | amount of quote
					["60", "120"], 2 * 60 usdc
					["80", "320"], 4 * 80 usdc
					["100", "600"], 6 * 100 usdc
			]
		},
	}
}
ZeroToOnePriceLevels: []Level{ // base -> quote | maker []bids
	{Price: 100, Quote: 2},
	{Price: 80, Quote: 4},
	{Price: 60, Quote: 10},
},
OneToZeroPriceLevels: []Level{ // quote -> base | maker []asks (1/price)
	{Price: 1.0 / 60, Quote: 120},
	{Price: 1.0 / 80, Quote: 320},
	{Price: 1.0 / 100, Quote: 600},
},
[base   | quote ] = [ETH | USDC] ("pair": "ETH/USDC")
[token0 | token1] = [ETH | USDC] (0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab < 0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e)
[token0 | token1] = [base   | quote ] = [ETH | USDC]
0to1 =taker amountIn base -> quote | maker buy base | use buyBook (1 base = ? quote)
1to0 =taker amountIn quote -> base | make sell base | use 1/sellBook (1 quote = ? base)
*/

var entityPool = entity.Pool{
	Address:  "dexalot_0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab_0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
	Exchange: "dexalot",
	Type:     "dexalot",
	Reserves: []string{"", ""},
	Tokens: []*entity.PoolToken{
		{Address: "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab", Decimals: 18, Swappable: true},
		{Address: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", Decimals: 6, Swappable: true},
	},

	Extra: "{\"0to1\":[" +
		"{\"q\":2,\"p\":100}," +
		"{\"q\":4,\"p\":80}," +
		"{\"q\":6,\"p\":60}" +
		"]," +
		"\"1to0\":[" +
		"{\"q\":120,\"p\":0.01666666667}," +
		"{\"q\":320,\"p\":0.0125}," +
		"{\"q\":600,\"p\":0.01}" +
		"]}",
}

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.Equal(t, "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab", poolSimulator.Token0.Address)
	assert.Equal(t, "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", poolSimulator.Token1.Address)
	assert.NotNil(t, poolSimulator.ZeroToOnePriceLevels)
	assert.NotNil(t, poolSimulator.OneToZeroPriceLevels)
	assert.Equal(t, []string{"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"},
		poolSimulator.CanSwapTo("0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"))
	assert.Equal(t, []string{"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"},
		poolSimulator.CanSwapFrom("0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"))
	assert.Equal(t, []string{"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"},
		poolSimulator.CanSwapTo("0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"))
	assert.Equal(t, []string{"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"},
		poolSimulator.CanSwapFrom("0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"))
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)

	tests := []struct {
		name              string
		amountIn          *big.Int
		expectedAmountOut string
		expectedErr       error
	}{
		{
			name:              "[1to0] it should return correct amountOut when amountIn = levels[0].Quote",
			amountIn:          big.NewInt(120000000),
			expectedAmountOut: "2", // ask["60", "120"] | 1to0[0.01666666667, 120] -> 120usdc = 2ETH
		},
		{
			name:              "[1to0] it should return correct amountOut when amountIn = levels[1].Quote",
			amountIn:          big.NewInt(320000000),
			expectedAmountOut: "4", // ask["80" "320"] | 1to0[0.0125, 320] -> 320usdc = 4ETH
		},
		{
			name:              "[1to0] it should return correct amountOut when amountIn between levels[0] and levels[1] quote",
			amountIn:          big.NewInt(200000000),
			expectedAmountOut: "3", // [0.01666666667, 120] [0.0125, 320] | [0.01666666667 + ((0.0125 - 0.01666666667) * (200-120) / (320-120))] * 200 = 0.015 * 200
		},
		{
			name:        "[1to0] it should return error when swap lower than level 0", //
			amountIn:    big.NewInt(120000000 - 1),
			expectedErr: ErrAmountInIsLessThanLowestPriceLevel,
		},
		{
			name:        "[1to0] it should return error when swap higher than total level", //
			amountIn:    big.NewInt(600000001),
			expectedErr: ErrAmountInIsGreaterThanHighestPriceLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
					Amount: tc.amountIn,
				},
				TokenOut: "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab",
			}
			tokenIn, tokenOut, levels := poolSimulator.Token0, poolSimulator.Token1, poolSimulator.ZeroToOnePriceLevels
			if params.TokenAmountIn.Token == poolSimulator.Info.Tokens[1] {
				tokenIn, tokenOut, levels = poolSimulator.Token1, poolSimulator.Token0, poolSimulator.OneToZeroPriceLevels
			}
			_, resultFloat, err := poolSimulator.swap(params.TokenAmountIn.Amount, tokenIn, tokenOut, "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab", "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", levels)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedAmountOut, resultFloat)
			}
		})
	}
}
