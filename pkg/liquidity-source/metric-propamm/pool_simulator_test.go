package metricpropamm

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator(t *testing.T) {
	file, err := os.ReadFile("./pool_state_test.json")
	assert.NoError(t, err)

	var poolState PairData

	err = json.Unmarshal(file, &poolState)
	assert.NoError(t, err)

	extra, reserves, err := convertPoolState(poolState, &Config{MaxAge: 60, IsV2: true})
	assert.NoError(t, err)

	WETH := "0x4200000000000000000000000000000000000006"
	USDC := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

	poolSimulator := &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  "0xa07938ea73e9d8eb23535816d073c2731ea24946",
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{WETH, USDC},
			Reserves: []*big.Int{bignumber.NewBig(reserves[0]), bignumber.NewBig(reserves[1])},
		},
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

func TestCloneState(t *testing.T) {
	t.Parallel()
	file, err := os.ReadFile("./pool_state_test.json")
	require.NoError(t, err)

	var pairData PairData
	require.NoError(t, json.Unmarshal(file, &pairData))

	extra, reserves, err := convertPoolState(pairData, &Config{MaxAge: 3600, IsV2: true})
	require.NoError(t, err)

	WETH := "0x4200000000000000000000000000000000000006"
	USDC := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

	p := &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  "0xa07938ea73e9d8eb23535816d073c2731ea24946",
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{WETH, USDC},
			Reserves: []*big.Int{bignumber.NewBig(reserves[0]), bignumber.NewBig(reserves[1])},
		},
		poolTimestamp: time.Now().Unix(),
		extra:         extra,
		decimalsDiff:  12,
	}

	testutil.TestCloneState(t, p, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  WETH,
			Amount: bignumber.NewBig("1000000000000000000"), // 1 WETH
		},
		TokenOut: USDC,
	}, nil)
}

// Each Metric extension adds swap gas: dropping the per-extension term makes quoted gas too low.
func TestGasPerExtension(t *testing.T) {
	file, err := os.ReadFile("./pool_state_test.json")
	require.NoError(t, err)
	var state PairData
	require.NoError(t, json.Unmarshal(file, &state))
	extra, reserves, err := convertPoolState(state, &Config{MaxAge: 60, IsV2: true})
	require.NoError(t, err)

	const weth, usdc = "0x4200000000000000000000000000000000000006", "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	gas := func(typ string, exts int) int64 {
		extra.Extensions = exts
		sim := &PoolSimulator{
			Info: pool.PoolInfo{
				Type:     typ,
				Tokens:   []string{weth, usdc},
				Reserves: []*big.Int{bignumber.NewBig(reserves[0]), bignumber.NewBig(reserves[1])},
			},
			poolTimestamp: time.Now().Unix(),
			extra:         extra,
			decimalsDiff:  12,
		}
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: bignumber.NewBig("10000000000000000")},
			TokenOut:      usdc,
		})
		require.NoError(t, err)
		return res.Gas
	}

	// literals on purpose: a changed constant must be a conscious edit
	require.Equal(t, int64(159433), gas(DexType, 0))
	require.Equal(t, int64(18650*3), gas(DexType, 3)-gas(DexType, 0))
}

// Indexing builds simulators with Opts.StaleCheck=false and must still quote from an aged pool.
func TestFactoryStaleCheck(t *testing.T) {
	file, err := os.ReadFile("./pool_state_test.json")
	require.NoError(t, err)
	var state PairData
	require.NoError(t, json.Unmarshal(file, &state))
	extra, reserves, err := convertPoolState(state, &Config{MaxAge: 60, IsV2: true})
	require.NoError(t, err)
	extraJSON, err := json.Marshal(extra)
	require.NoError(t, err)

	weth, usdc := "0x4200000000000000000000000000000000000006", "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	ep := entity.Pool{
		Address: "0xa07938ea73e9d8eb23535816d073c2731ea24946", Exchange: DexType, Type: DexType,
		Reserves: entity.PoolReserves{reserves[0], reserves[1]}, Extra: string(extraJSON), StaticExtra: "{}",
		Tokens:    []*entity.PoolToken{{Address: weth, Decimals: 18, Swappable: true}, {Address: usdc, Decimals: 6, Swappable: true}},
		Timestamp: time.Now().Unix() - 3600,
	}
	for _, staleCheck := range []bool{false, true} {
		sim, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: staleCheck}})
		require.NoError(t, err)
		_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: bignumber.NewBig("10000000000000000")}, TokenOut: usdc,
		})
		if staleCheck {
			assert.ErrorIs(t, err, ErrStalePoolData)
		} else {
			assert.NoError(t, err)
		}
	}
}

func convertPoolState(d PairData, c *Config) (Extra, []string, error) {
	return Extra{
		InitBid: bignumber.NewBig(d.Bid), InitAsk: bignumber.NewBig(d.Ask), QuoteAvailable: d.QuoteAvailable,
		MaxAge: c.MaxAge, IsV2: c.IsV2, Bids: ConvertBins(d.Depth.Bids), Asks: ConvertBins(d.Depth.Asks),
	}, []string{d.TotalToken0Available, d.TotalToken1Available}, nil
}
