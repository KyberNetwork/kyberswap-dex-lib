package plain

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/meta"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copy from curve-meta

func TestCalcAmountOutAsBasePool(t *testing.T) {
	// test data from https://etherscan.io/address/0x0f9cb53ebe405d49a0bbdbd291a65ff571bc83e1#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"Am", 1000, "Bm", 31},
		{"Bm", 2, "Am", 61},

		{"Am", 1000, "A", 31},
		{"Am", 1000000000000000, "B", 32},
		{"Am", 1000000000000000, "C", 32},

		{"A", 10, "Am", 277},
		{"A", 1000000000000000, "B", 999},
		{"A", 1000000000000000, "C", 1000},

		{"B", 3, "Am", 92474249943422}, // get_dy_underlying return 92475148432038, but the actual swap is different
		{"B", 1, "A", 999909687790},
		{"B", 100, "C", 100},

		{"C", 2, "Am", 61627616659617}, // get_dy_underlying return 61628215439376, but the actual swap is different
		{"C", 3, "A", 2998664269827},
		{"C", 30, "B", 29},
	}
	base, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243", "352290453972395231054279357"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}, {Address: "C", Decimals: 6}},
		Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
		StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\"}",
	})
	require.Nil(t, err)

	p, err := meta.NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
		StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
	}, base)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			meta := p.GetMetaInfo(tc.in, tc.out)
			// if any side is from underlying base pool then need to use underlying call
			isUnderlying := !strings.HasSuffix(tc.in, "m") || !strings.HasSuffix(tc.out, "m")
			assert.Equal(t, isUnderlying, meta.(curve.Meta).Underlying)
		})
	}
}

func TestUpdateBalanceAsBasePool(t *testing.T) {
	// test data from https://etherscan.io/address/0x0f9cb53ebe405d49a0bbdbd291a65ff571bc83e1#readContract
	testcases := []struct {
		in               string
		inAmount         int64
		out              string
		expectedBalances []string
	}{
		{"Am", 1000, "Bm", []string{"4763102571534863472314821", "15272752439110430673250"}},
		{"Am", 1000000000000000, "B", []string{"4763102572534863472314821", "15272752407518134109468"}},
		{"C", 2, "Am", []string{"4763102572473232773721712", "15272752409466747992850"}},
	}
	base, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"93650900813860355891321787", "92392098150103", "175345980953129", "352170672490633463630226070"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}, {Address: "C", Decimals: 6}},
		Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
		StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\"}",
	})
	require.Nil(t, err)

	p, err := meta.NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
		StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
	}, base)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: amountIn,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  amountIn,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})

			for i, expBalance := range tc.expectedBalances {
				assert.Equal(t, bignumber.NewBig10(expBalance), p.Info.Reserves[i])
			}
		})
	}
}
