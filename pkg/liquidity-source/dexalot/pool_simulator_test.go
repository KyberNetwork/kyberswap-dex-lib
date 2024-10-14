package dexalot

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

/*
0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab // eth
0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e // usdc

	{
	    "data": {
	        "buyBook": [
	            {
	                "prices": "2420200000,2420100000,2418500000,2417700000,2416500000,2415900000,2411700000,2411600000,2410500000,2406900000,2401300000,2370500000,2363100000,2265400000,2263400000,2261500000,1286500000,186500000",
	                "quantities": "15000000000000000000,384190000000000000,1067430000000000000,4962360000000000000,2112810000000000000,15000000000000000000,15000000000000000000,3373380000000000000,41900000000000000,15000000000000000000,5529320000000000000,47320000000000000,47460000000000000,111080000000000000,57440000000000000,57480000000000000,104160000000000000,718500000000000000",
	                "baseCumulative": "15000000000000000000,15384190000000000000,16451620000000000000,21413980000000000000,23526790000000000000,38526790000000000000,53526790000000000000,56900170000000000000,56942070000000000000,71942070000000000000,77471390000000000000,77518710000000000000,77566170000000000000,77677250000000000000,77734690000000000000,77792170000000000000,77896330000000000000,78614830000000000000",
	                "quoteCumulative": "36303000000,37232778219,39814357674,51811855446,56917460811,93155960811,129331460811,137466704019,137567703969,173671203969,186948760085,187060932145,187173084871,187424725503,187554735199,187684726219,187818728059,187952728309",
	                "quoteTotal": "36303000000,929778219,2581579455,11997497772,5105605365,36238500000,36175500000,8135243208,100999950,36103500000,13277556116,112172060,112152726,251640632,130009696,129991020,134001840,134000250"
	            }
	        ],
	        "sellBook": [
	            {
	                "prices": "2420700000,2421200000,2422100000,2422200000,2422300000,2423600000,2423900000,2424600000,2426400000,2429900000,2431200000,2438100000,3218000000,3511700000,3512700000,3513400000,3517100000,3530400000,3561000000,3718000000",
	                "quantities": "15000000000000000000,439570000000000000,1651320000000000000,15000000000000000000,1124250000000000000,1650330000000000000,1905360000000000000,823760000000000000,15000000000000000000,2906620000000000000,15000000000000000000,6171670000000000000,10000000000000000,37040000000000000,46450000000000000,37040000000000000,40820000000000000,40790000000000000,44930000000000000,7690000000000000",
	                "baseCumulative": "15000000000000000000,15439570000000000000,17090890000000000000,32090890000000000000,33215140000000000000,34865470000000000000,36770830000000000000,37594590000000000000,52594590000000000000,55501210000000000000,70501210000000000000,76672880000000000000,76682880000000000000,76719920000000000000,76766370000000000000,76803410000000000000,76844230000000000000,76885020000000000000,76929950000000000000,76937640000000000000",
	                "quoteCumulative": "36310500000,37374786884,41374449056,77707449056,80430719831,84430459619,89048861723,91046150219,127442150219,134504946157,170972946157,186020094784,186052274784,186182348152,186345513067,186475649403,186619217425,186763222441,186923218171,186951809591",
	                "quoteTotal": "36310500000,1064286884,3999662172,36333000000,2723270775,3999739788,4618402104,1997288496,36396000000,7062795938,36468000000,15047148627,32180000,130073368,163164915,130136336,143568022,144005016,159995730,28591420"
	            }
	        ],
	        "spread": "500000",
	        "spreadBasis": "2065731.57883864570638",
	        "midPrice": "2420450000",
	        "rewardInfo": {
	            "min": "2396000000",
	            "max": "2444900000",
	            "maxSpread": "0.003000000000000000",
	            "maxDepth": "0.010000000000000000"
	        }
	    },
	    "type": "orderBooks",
	    "pair": "ETH/USDC",
	    "decimal": 1
	}

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
		"{\"q\":15,\"p\":2420.2}," +
		"{\"q\":0.38419,\"p\":2420.1}," +
		"{\"q\":1.06743,\"p\":2418.5}" +
		"]," +
		"\"1to0\":[" +
		"{\"q\":36310.5,\"p\":0.0004131036477}," +
		"{\"q\":1064.286884,\"p\":0.000413018338}," +
		"{\"q\":3999.662172,\"p\":0.0004128648693}" +
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
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          big.NewInt(2420700000),
			expectedAmountOut: "1",
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          big.NewInt(4841400000),
			expectedAmountOut: "2",
		},
		{
			name:              "it should return correct amountOut when swap in levels",
			amountIn:          big.NewInt(36320500000),
			expectedAmountOut: "15.00413018", // 36310.5 * 0.0004131036477 + 10 * 0.000413018338
		},
		{
			name:        "it should return error when swap higher than total level", //
			amountIn:    big.NewInt(50_000_000_000),
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
			_, resultFloat, err := poolSimulator.swap(params.TokenAmountIn.Amount, tokenIn, tokenOut, levels)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, tc.expectedAmountOut, resultFloat)
			}
		})
	}
}
