package base

import (
	"fmt"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmtOut(t *testing.T) {
	t.Parallel()
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	// 	call balances, totalSupply to get pool params
	// 	call get_dy to get amount out
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
		expectedFeeAmount int64
	}{
		{"A", 5000, "B", 4998, 1},
		{"A", 50000, "B", 49986, 15},
		{"B", 50000, "A", 49983, 14},
		{"B", 51000, "A", 50982, 15},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, 0, len(p.CanSwapTo("LP")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmtOutResult, error) {
				var in uint256.Int
				in.SetUint64(uint64(tc.inAmount))
				return p.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: tc.in, Amount: in},
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, uint256.NewInt(uint64(tc.expectedOutAmount)).String(), out.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			assert.Equal(t, uint256.NewInt(uint64(tc.expectedFeeAmount)).String(), out.Fee.Amount.String())
		})
	}
}

func TestCalcAmtOut_interpolate_from_initialA_and_futureA(t *testing.T) {
	t.Parallel()
	// if A is getting ramped up then it should interpolate A correctly
	// 100k at zero to 200k at now*2, so now should be 150k, so the same as the contract above -> get expected output from contract get_dy
	now := time.Now().Unix()
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"futureATime\": %v}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			100000, 200000,
			now*2),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(t, err)

	out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmtOutResult, error) {
		var in uint256.Int
		in.SetUint64(510000)
		return p.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: "A", Amount: in},
			TokenOut:      "B",
		})
	})
	require.Nil(t, err)
	assert.Equal(t, "509863", out.TokenAmountOut.Amount.String())
	assert.Equal(t, "153", out.Fee.Amount.String())
}

// TestCalcAmtOutGetDOverflow verifies that a pool state where D_P*D would overflow uint256 during getD returns an
// error instead of silently producing a result. The on-chain contract reverts there, so the quote must fail too.
func TestCalcAmtOutGetDOverflow(t *testing.T) {
	t.Parallel()
	// 3-token pool. Token C has a reserve of 1e36 (18-decimal, rate 1e18).
	// xp[2] ≈ 1e36 → D ≈ 1e36 → at j=1 of the D_P loop: D_P*D ≈ 3.3e90, far above uint256 max (1.16e77).
	p, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{
			"100000000000000000",                    // A: 1e17
			"100000000000000000",                    // B: 1e17
			"1000000000000000000000000000000000000", // C: 1e36 — degenerate
			"1000000000000000000000000000000000000", // LP supply
		},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       `{"swapFee":"3000000","adminFee":"5000000000","initialA":"150000","futureA":"150000"}`,
		StaticExtra: `{"lpToken":"LP","aPrecision":"1","precisionMultipliers":["1","1","1"],"rates":["1000000000000000000","1000000000000000000","1000000000000000000"]}`,
	})
	require.NoError(t, err)

	pairs := [][2]string{{"A", "B"}, {"A", "C"}, {"B", "C"}}
	for _, pair := range pairs {
		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmtOutResult, error) {
			var in uint256.Int
			in.SetUint64(1e15)
			return p.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: pair[0], Amount: in},
				TokenOut:      pair[1],
			})
		})
		require.ErrorIs(t, err, ErrDDoesNotConverge, "expected overflow-detected error for %s→%s", pair[0], pair[1])
	}
}

// TestCalcAmtOutLegacyGolden pins CalcAmtOut to the big.Int CalcAmountOut results captured before the uint256 port.
// Each row is {tokenIn, amountIn, tokenOut, wantOut, wantFee}. An empty wantOut means the swap must fail.
func TestCalcAmtOutLegacyGolden(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"swapFee":"3000000","adminFee":"5000000000","initialA":"150000","futureA":"150000"}`,
		StaticExtra: `{"lpToken":"LP","aPrecision":"100","precisionMultipliers":["1000000000000","1000000000000"],` +
			`"rates":["1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.NoError(t, err)
	for _, row := range [][5]string{
		{"A", "2", "B", "1", "0"},
		{"B", "2", "A", "1", "0"},
		{"A", "10", "B", "9", "0"},
		{"B", "10", "A", "9", "0"},
		{"A", "100", "B", "99", "0"},
		{"B", "100", "A", "99", "0"},
		{"A", "1000", "B", "999", "0"},
		{"B", "1000", "A", "999", "0"},
		{"A", "5000", "B", "4998", "1"},
		{"B", "5000", "A", "4998", "1"},
		{"A", "50000", "B", "49986", "15"},
		{"B", "50000", "A", "49983", "14"},
		{"A", "51000", "B", "50986", "15"},
		{"B", "51000", "A", "50982", "15"},
		{"A", "500000", "B", "499866", "150"},
		{"B", "500000", "A", "499830", "149"},
		{"A", "5000000", "B", "4998519", "1500"},
		{"B", "5000000", "A", "4998160", "1499"},
	} {
		var in uint256.Int
		require.NoError(t, in.SetFromDecimal(row[1]))
		res, err := p.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: row[0], Amount: in}, TokenOut: row[2]})
		require.NoError(t, err, "%v", row)
		require.Equal(t, row[3], res.TokenAmountOut.Amount.Dec(), "%v", row)
		require.Equal(t, row[4], res.Fee.Amount.Dec(), "%v", row)
	}
}

func BenchmarkCalcAmtOut(b *testing.B) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(b, err)

	var in uint256.Int
	in.SetUint64(5000)
	for range b.N {
		_, err := p.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: "A", Amount: in},
			TokenOut:      "B",
		})
		require.Nil(b, err)
	}
}

func TestGetDyVirtualPrice(t *testing.T) {
	t.Parallel()
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	testcases := []struct {
		i      int
		j      int
		dx     string
		expOut string
	}{
		{0, 1, "100000000000000", "140254485"},
		{0, 1, "1000000000000", "140254485"},
		{0, 1, "100000000", "99936588"},
		{0, 1, "100002233", "99938816"},
		{1, 0, "20000", "19982"},
		{1, 0, "3000200", "2997456"},
		{1, 0, "88001800", "69067695"},
		{1, 0, "100000000000000", "69244498"},
	}
	poolRedis := `{
		"address": "0x1005f7406f32a61bd760cfa14accd2737913d546",
		"reserveUsd": 209.42969262729198,
		"amplifiedTvl": 209.42969262729198,
		"exchange": "curve",
		"type": "curve-base",
		"timestamp": 1705393976,
		"reserves": ["69265278", "140296574", "208111994100559113335"],
		"tokens": [
			{ "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "weight": 1, "swappable": true },
			{ "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "weight": 1, "swappable": true }
		],
		"extra": "{\"initialA\":\"150000\",\"futureA\":\"150000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"3000000\",\"adminFee\":\"5000000000\"}",
		"staticExtra": "{\"lpToken\":\"0x1005f7406f32a61bd760cfa14accd2737913d546\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}"
	}`
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	v, dCached, err := p.GetVirtualPrice()
	require.Nil(t, err)
	assert.Equal(t, "1006923185919753102", v.Dec())

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			var dx uint256.Int
			require.NoError(t, dx.SetFromDecimal(tc.dx))
			dy, err := testutil.MustConcurrentSafe(t, func() (*uint256.Int, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, &dx, nil)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, tc.expOut, dy.Dec())

			// test using cached D
			dy, err = testutil.MustConcurrentSafe(t, func() (*uint256.Int, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, &dx, dCached)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, tc.expOut, dy.Dec())
		})
	}
}

func TestCloneState(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"swapFee":"3000000","adminFee":"5000000000","initialA":"150000","futureA":"150000"}`,
		StaticExtra: `{"lpToken":"LP","aPrecision":"100","precisionMultipliers":["1000000000000","1000000000000"],` +
			`"rates":["1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.NoError(t, err)

	var in uint256.Int
	in.SetUint64(5000)
	testutil.TestCloneStateIPoolSim(t, p, pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: "A", Amount: in},
		TokenOut:      "B",
	})
}
