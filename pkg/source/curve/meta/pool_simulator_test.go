package meta

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
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

		{"B", 3, "Am", 92475148432038},
		{"B", 1, "A", 999909687790},
		{"B", 100, "C", 100},

		{"C", 2, "Am", 61628215439376},
		{"C", 3, "A", 2998664269827},
		{"C", 30, "B", 29},
	}
	basePool, err := base.NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243",
			"352290453972395231054279357"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"initialA":"5000","futureA":"2000","initialATime":1653559305,"futureATime":1654158027,"swapFee":"1000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LPBase","aPrecision":"1","precisionMultipliers":["1","1000000000000","1000000000000"],"rates":["1000000000000000000","1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.Nil(t, err)
	basePoolMap := map[string]pool.IPoolSimulator{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": basePool}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       `{"initialA":"10000","futureA":"25000","initialATime":1649327847,"futureATime":1649925962,"swapFee":"4000000","adminFee":"0"}`,
		StaticExtra: `{"lpToken":"LPMeta","basePool":"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7","rateMultiplier":"1000000000000000000","aPrecision":"100","underlyingTokens":["0x674c6ad92fd080e4004b2312b45f796a192d27a0","0x6b175474e89094c44da98b954eedeac495271d0f","0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","0xdac17f958d2ee523a2206206994597c13d831ec7"],"precisionMultipliers":["1","1"],"rates":["",""]}`,
	}, basePoolMap)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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

func TestCalcAmountOut_Underflow(t *testing.T) {
	t.Parallel()
	// test data from 0xf07d553b195080f84f582e88ecdd54baa122b279
	testcases := []struct {
		in       string
		inAmount int64
		out      string
	}{
		{"Am", 1, "A"},
	}
	basePool, err := plain.NewPoolSimulator(entity.Pool{
		Exchange:    "curve-stable-plain",
		Type:        "curve-stable-plain",
		Reserves:    entity.PoolReserves{"4328477915799", "2193973068000", "6401362516550506952404697"},
		Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 6}, {Address: "B", Decimals: 6}},
		Extra:       `{"InitialA":"100000","FutureA":"200000","InitialATime":1673284886,"FutureATime":1673889683,"SwapFee":"100000","AdminFee":"5000000000"}`,
		StaticExtra: `{"LpToken":"LPBase","APrecision":"100","IsNativeCoin":[false,false]}`,
	})
	require.Nil(t, err)
	basePoolMap := map[string]pool.IPoolSimulator{"LPBase": basePool}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "curve",
		Type:        "curve-meta",
		Reserves:    entity.PoolReserves{"107979258293367959147", "47194924911735952439", "4715249265933991444"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       `{"initialA":"20000","futureA":"20000","initialATime":0,"futureATime":0,"swapFee":"4000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LPMeta","basePool":"LPBase","rateMultiplier":"1000000000000000000","aPrecision":"100","underlyingTokens":["Am","A","B"],"precisionMultipliers":["1","1"],"rates":["",""]}`,
	}, basePoolMap)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			assert.Error(t, err)
		})
	}
}

func TestSwappable(t *testing.T) {
	t.Parallel()

	basePool, err := base.NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243",
			"352290453972395231054279357"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"initialA":"5000","futureA":"2000","initialATime":1653559305,"futureATime":1654158027,"swapFee":"1000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LPBase","aPrecision":"1","precisionMultipliers":["1","1000000000000","1000000000000"],"rates":["1000000000000000000","1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.Nil(t, err)
	basePoolMap := map[string]pool.IPoolSimulator{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": basePool}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       `{"initialA":"10000","futureA":"25000","initialATime":1649327847,"futureATime":1649925962,"swapFee":"4000000","adminFee":"0"}`,
		StaticExtra: `{"lpToken":"LPMeta","basePool":"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7","rateMultiplier":"1000000000000000000","aPrecision":"100","underlyingTokens":["0x674c6ad92fd080e4004b2312b45f796a192d27a0","0x6b175474e89094c44da98b954eedeac495271d0f","0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","0xdac17f958d2ee523a2206206994597c13d831ec7"],"precisionMultipliers":["1","1"],"rates":["",""]}`,
	}, basePoolMap)
	require.Nil(t, err)

	// lpToken can't be swapped to anything
	assert.Equal(t, 0, len(p.CanSwapTo("LPMeta")))
	assert.Equal(t, 0, len(p.CanSwapTo("LPBase")))

	assert.Equal(t, 0, len(p.CanSwapTo("XXX")))

	// 1st meta token can be swapped to anything
	assert.Equal(t, []string{"Bm", "A", "B", "C"}, p.CanSwapTo("Am"))

	// last meta token can't be swapped to anything other than the 1st one
	assert.Equal(t, []string{"Am"}, p.CanSwapTo("Bm"))

	// base token can be swapped to anything other than the last meta token
	assert.Equal(t, []string{"Am"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"Am"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"Am"}, p.CanSwapTo("C"))

	errorcases := []struct {
		in  string
		out string
	}{
		{"LPMeta", "Am"},
		{"LPMeta", "A"},
		{"LPBase", "Am"},
		{"LPBase", "A"},
		{"Bm", "A"},
		{"Bm", "B"},
		{"Bm", "C"},
		{"A", "Bm"},
		{"B", "Bm"},
		{"C", "Bm"},

		{"XXX", "A"},
		{"Am", "YYY"},
	}

	for idx, tc := range errorcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(100000000)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.NotNil(t, err)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
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
	basePool, err := base.NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"93650900813860355891321787", "92392098150103", "175345980953129",
			"352170672490633463630226070"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"initialA":"5000","futureA":"2000","initialATime":1653559305,"futureATime":1654158027,"swapFee":"1000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LPBase","aPrecision":"1","precisionMultipliers":["1","1000000000000","1000000000000"],"rates":["1000000000000000000","1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.Nil(t, err)
	basePoolMap := map[string]pool.IPoolSimulator{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": basePool}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       `{"initialA":"10000","futureA":"25000","initialATime":1649327847,"futureATime":1649925962,"swapFee":"4000000","adminFee":"0"}`,
		StaticExtra: `{"lpToken":"LPMeta","basePool":"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7","rateMultiplier":"1000000000000000000","aPrecision":"100","underlyingTokens":["0x674c6ad92fd080e4004b2312b45f796a192d27a0","0x6b175474e89094c44da98b954eedeac495271d0f","0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","0xdac17f958d2ee523a2206206994597c13d831ec7"],"precisionMultipliers":["1","1"],"rates":["",""]}`,
	}, basePoolMap)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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

func BenchmarkGetDyUnderlying(b *testing.B) {

	// {"Am", 1000, "A", 31},
	basePool, err := base.NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243",
			"352290453972395231054279357"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"initialA":"5000","futureA":"2000","initialATime":1653559305,"futureATime":1654158027,"swapFee":"1000000","adminFee":"5000000000"}`,
		StaticExtra: `{"lpToken":"LPBase","aPrecision":"1","precisionMultipliers":["1","1000000000000","1000000000000"],"rates":["1000000000000000000","1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.Nil(b, err)
	basePoolMap := map[string]pool.IPoolSimulator{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": basePool}

	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
		Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
		Extra:       `{"initialA":"10000","futureA":"25000","initialATime":1649327847,"futureATime":1649925962,"swapFee":"4000000","adminFee":"0"}`,
		StaticExtra: `{"lpToken":"LPMeta","basePool":"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7","rateMultiplier":"1000000000000000000","aPrecision":"100","underlyingTokens":["0x674c6ad92fd080e4004b2312b45f796a192d27a0","0x6b175474e89094c44da98b954eedeac495271d0f","0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","0xdac17f958d2ee523a2206206994597c13d831ec7"],"precisionMultipliers":["1","1"],"rates":["",""]}`,
	}, basePoolMap)
	require.Nil(b, err)

	for i := 0; i < b.N; i++ {
		_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "B", Amount: big.NewInt(10)},
			TokenOut:      "A",
			Limit:         nil,
		})
		require.Nil(b, err)
	}
}

func TestRAISwap(t *testing.T) {
	now := time.Now().Unix()
	poolStr := "{\"address\":\"0x618788357d0ebd8a37e763adab3bc575d54c2c7d\",\"amplifiedTvl\":96473.18024615978,\"exchange\":\"curve\",\"type\":\"curve-meta\",\"timestamp\":1752141285,\"reserves\":[\"25207726126074011679635\",\"20333558078904652962161\",\"0\"],\"tokens\":[{\"address\":\"0x03ab458634910aad20ef5f1c8ee96f1d6ac54919\",\"symbol\":\"RAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x6c3f90f043a72fa612cbac8115ee7e52bde6e490\",\"symbol\":\"3Crv\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"initialA\\\":\\\"10000\\\",\\\"futureA\\\":\\\"10000\\\",\\\"initialATime\\\":0,\\\"futureATime\\\":0,\\\"swapFee\\\":\\\"4000000\\\",\\\"adminFee\\\":\\\"5000000000\\\",\\\"snappedRedemptionPrice\\\":3049991316778665711364455710}\",\"staticExtra\":\"{\\\"lpToken\\\":\\\"0x618788357d0ebd8a37e763adab3bc575d54c2c7d\\\",\\\"basePool\\\":\\\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\\\",\\\"rateMultiplier\\\":\\\"1000000000000000000\\\",\\\"aPrecision\\\":\\\"100\\\",\\\"underlyingTokens\\\":[\\\"0x03ab458634910aad20ef5f1c8ee96f1d6ac54919\\\",\\\"0x6b175474e89094c44da98b954eedeac495271d0f\\\",\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\",\\\"0xdac17f958d2ee523a2206206994597c13d831ec7\\\"],\\\"precisionMultipliers\\\":[\\\"1\\\",\\\"1\\\"],\\\"rates\\\":[\\\"\\\",\\\"\\\"]}\"}"
	basePoolStr := fmt.Sprintf("{\"address\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"amplifiedTvl\":167594991.2197165,\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1752069985,\"reserves\":[\"75000987250283023485540264\",\"60795598086384\",\"46881473180944\",\"175680474464184526040181476\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"symbol\":\"DAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"4000\\\",\\\"InitialATime\\\":%d,\\\"FutureATime\\\":%d,\\\"SwapFee\\\":\\\"1500000\\\",\\\"AdminFee\\\":\\\"10000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"1\\\",\\\"LpToken\\\":\\\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\\\",\\\"IsNativeCoin\\\":[false,false,false]}\",\"blockNumber\":22882112}", now, now+2000)

	var basePool entity.Pool
	_ = json.Unmarshal([]byte(basePoolStr), &basePool)

	var p entity.Pool
	_ = json.Unmarshal([]byte(poolStr), &p)

	basePoolSimulator, err := plain.NewPoolSimulator(basePool)
	assert.NoError(t, err)

	simulator, _ := NewPoolSimulator(p, map[string]pool.IPoolSimulator{
		"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": basePoolSimulator,
	})
	assert.NoError(t, err)

	DAI := "0x6b175474e89094c44da98b954eedeac495271d0f"
	RAI := "0x03ab458634910aad20ef5f1c8ee96f1d6ac54919"
	amountIn, _ := new(big.Int).SetString("24000000000000000000", 10)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  DAI,
			Amount: amountIn,
		},
		TokenOut: RAI,
	}

	res, err := simulator.CalcAmountOut(params)
	assert.NoError(t, err)
	assert.Equal(t, "8056187488661470351", res.TokenAmountOut.Amount.String())
}
