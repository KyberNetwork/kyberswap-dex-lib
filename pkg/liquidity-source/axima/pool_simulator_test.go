package axima

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator(t *testing.T) {
	file, err := os.ReadFile("./pool_state_test.json")
	assert.NoError(t, err)

	var aximaPoolState PairData

	err = json.Unmarshal(file, &aximaPoolState)
	assert.NoError(t, err)

	extra, reserves, err := convertAximaPoolState(aximaPoolState, &Config{MaxAge: 60, IsV2: true})
	assert.NoError(t, err)

	WETH := "0x4200000000000000000000000000000000000006"
	USDC := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

	poolSimulator := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  "0xa07938ea73e9d8eb23535816d073c2731ea24946",
			Exchange: "axima",
			Type:     "axima",
			Tokens:   []string{WETH, USDC},
			Reserves: []*big.Int{bignumber.NewBig(reserves[0]), bignumber.NewBig(reserves[1])},
		}},
		poolTimestamp: time.Now().Unix(),
		extra:         extra,
		decimalsDiff:  12,
	}

	testCases := []struct {
		params            pool.CalcAmountOutParams
		expectedAmountOut string
		expectedErr       error
	}{
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("10000000000000000"), // 0.01 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "20634466",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("100000000000000000"), // 0.1 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "206343731",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("1000000000000000000"), // 1 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "2063376444",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("5000000000000000000"), // 5 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "10316183125",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("10000000000000000000"), // 10 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "20630685281",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  USDC,
					Amount: bignumber.NewBig("100000000"), // 100 USDC
				},
				TokenOut: WETH,
			},
			expectedAmountOut: "48458452787303089",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  USDC,
					Amount: bignumber.NewBig("500000000"), // 500 USDC
				},
				TokenOut: WETH,
			},
			expectedAmountOut: "242291145383660839",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  USDC,
					Amount: bignumber.NewBig("1000000000"), // 1000 USDC
				},
				TokenOut: WETH,
			},
			expectedAmountOut: "484579494414231635",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  USDC,
					Amount: bignumber.NewBig("5000000000"), // 5000 USDC
				},
				TokenOut: WETH,
			},
			expectedAmountOut: "2422793122048649375",
			expectedErr:       nil,
		},
		{
			params: pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  WETH,
					Amount: bignumber.NewBig("30000000000000000000"), // 30 WETH
				},
				TokenOut: USDC,
			},
			expectedAmountOut: "",
			expectedErr:       ErrInsufficientLiquidity,
		},
	}

	for _, tc := range testCases {
		result, err := poolSimulator.CalcAmountOut(tc.params)
		assert.Equal(t, tc.expectedErr, err)
		if tc.expectedErr == nil {
			assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount.String())
		}
	}
}
