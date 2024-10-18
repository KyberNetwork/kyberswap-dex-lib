package bebop

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

var (
	entityPool1 = entity.Pool{
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
	entityPool2 = entity.Pool{
		Address:  "bebop_0x7a58c0be72be218b41c608b7fe7c5bb630736c71_0xdac17f958d2ee523a2206206994597c13d831ec7",
		Exchange: "bebop",
		Type:     "bebop",
		Reserves: []string{"1740906659949395367165952", "0"},
		Tokens: []*entity.PoolToken{
			{Address: "0x7a58c0be72be218b41c608b7fe7c5bb630736c71", Decimals: 18, Swappable: true},
			{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Swappable: true},
		},
		Extra: "{\"0to1\":[{\"p\":0.07595710380734046,\"q\":48163.55734890566},{\"p\":0.07593455722483192,\"q\":52993.32482126679},{\"p\":0.07591500784305961,\"q\":58302.68056018904},{\"p\":0.0758988654988538,\"q\":64137.646452324116},{\"p\":0.07587521441482821,\"q\":70558.7131658853},{\"p\":0.07585281405928816,\"q\":77614.58448247385},{\"p\":0.07582033645336315,\"q\":16049.424434351078},{\"p\":0.07580864440234752,\"q\":85391.1771552143},{\"p\":0.07575182579202616,\"q\":93949.7020470411},{\"p\":0.0756894119059153,\"q\":103355.92190388206},{\"p\":0.07560737490511166,\"q\":113709.06592087087},{\"p\":0.0755078631348937,\"q\":125095.00342213444},{\"p\":0.07539040528554969,\"q\":137608.56064368493},{\"p\":0.07523516462137005,\"q\":151388.32489547844},{\"p\":0.07504191048482589,\"q\":166548.79293229152},{\"p\":0.07480759629995394,\"q\":183214.78020295245},{\"p\":0.07450720327739284,\"q\":192825.3995604493}],\"1to0\":[]}",
	}
)

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool1)
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
	poolSimulator, err := NewPoolSimulator(entityPool1)
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
				Limit:    NewLimit(nil),
			}

			result, err := poolSimulator.CalcAmountOut(params)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_GetAmountOut2(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool2)
	assert.NoError(t, err)

	tests := []struct {
		name              string
		updateLimit       bool
		amountIn          *big.Int
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:              "it should return correct amountOut when swap in levels",
			updateLimit:       false,
			amountIn:          bigIntFromString("100000000000000000000"),
			expectedAmountOut: bigIntFromString("7595710"),
		},
		{
			name:              "it should return not enough inventory",
			updateLimit:       true,
			amountIn:          bigIntFromString("100000000000000000000"),
			expectedAmountOut: bigIntFromString("7595710"),
			expectedErr:       pool.ErrNotEnoughInventory,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap one to zero
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0x7a58c0be72be218b41c608b7fe7c5bb630736c71",
					Amount: tc.amountIn,
				},
				TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
				Limit:    NewLimit(nil),
			}
			if tc.updateLimit {
				_, _, _ = params.Limit.UpdateLimit("", "", nil, nil)
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
