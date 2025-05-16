package saddle

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut_Saddle(t *testing.T) {
	t.Parallel()
	// test data from https://etherscan.io/address/0xa6018520eaacc06c30ff2e1b3ee2c7c22e64196a#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10000, "B", 10089},
		{"A", 10000, "C", 9999},
		{"C", 10000, "B", 10086},

		{"LP", 10000, "A", 10008},
		{"LP", 10000, "B", 10103},
		{"LP", 10000, "C", 10011},

		{"A", 10000, "LP", 9989},
		{"B", 10000, "LP", 9898},
		{"C", 10000, "LP", 9986},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907",
			"553429429583268691085"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       "{\"initialA\":\"48000\",\"futureA\":\"92000\",\"initialATime\":1652287436,\"futureATime\":1653655053,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\"]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A", "B", "C"}, p.CanSwapTo("LP"))
	assert.Equal(t, []string{"B", "C", "LP"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A", "C", "LP"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"A", "B", "LP"}, p.CanSwapTo("C"))

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
		})
	}
}

func TestCalcAmountOut_Nerve(t *testing.T) {
	t.Parallel()
	// test data from https://bscscan.com/address/0x146cd24dcc9f4eb224dfd010c5bf2b0d25afa9c0#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10000000, "B", 9947419},
		{"B", 10000000, "A", 10036781},
		{"LP", 10000000, "A", 10052106},
		{"B", 10000000, "LP", 9990133},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"20288190723295606376", "9812867150429539713", "29980929628444248071"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:       "{\"initialA\":\"10000\",\"futureA\":\"20000\",\"initialATime\":1620946481,\"futureATime\":1622245581,\"swapFee\":\"8000000\",\"adminFee\":\"9999999999\",\"defaultWithdrawFee\":\"0\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\"], \"totalSupply\": \"29980929628444248071\"}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A", "B"}, p.CanSwapTo("LP"))
	assert.Equal(t, []string{"B", "LP"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A", "LP"}, p.CanSwapTo("B"))

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
		})
	}
}

func TestCalcAmountOut_OneSwap(t *testing.T) {
	t.Parallel()
	// test data from https://bscscan.com/address/0x01c9475dbd36e46d1961572c8de24b74616bae9e#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 1000000, "B", 999945},
		{"A", 1000000, "D", 999766},
		{"C", 1000000, "B", 999587},

		{"A", 1000000, "LP", 985028},

		// simulation yield 1015194 because withdrawfee is 0, but here we're using defaultWithdrawFee 5000000 so will be lower
		{"LP", 1000000, "B", 1014686},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"339028421564024338437", "347684462442560871352", "423798212946198474118",
			"315249216225911580289", "1404290718401538825321"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}, {Address: "D"}},
		Extra:       "{\"initialA\":\"60000\",\"futureA\":\"60000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"1000000\",\"adminFee\":\"10000000000\",\"defaultWithdrawFee\":\"5000000\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\",\"1\"]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A", "B", "C", "D"}, p.CanSwapTo("LP"))
	assert.Equal(t, []string{"B", "C", "D", "LP"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A", "C", "D", "LP"}, p.CanSwapTo("B"))

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
		})
	}
}

func TestCalcAmountOut_IronStable(t *testing.T) {
	t.Parallel()
	// test data from https://polygonscan.com/address/0x837503e8a8753ae17fb8c8151b8e6f586defcb57#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 1000000, "B", 999660},
		{"A", 1000000, "C", 999784329608585573},
		{"C", 10000000000000000, "B", 9996},

		{"A", 100, "LP", 98972249301594},

		// same as oneswap above, lower than simulation because of defaultWithdrawFee
		{"LP", 10000000000000000, "B", 10095},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"233518765839", "198509040315", "228986742536043517345011",
			"654251953025609178732174"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       "{\"initialA\":\"18000\",\"futureA\":\"120000\",\"initialATime\":1627094541,\"futureATime\":1627699238,\"swapFee\":\"2000000\",\"adminFee\":\"10000000000\", \"defaultWithdrawFee\":\"5000000\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\",\"1\"]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A", "B", "C"}, p.CanSwapTo("LP"))
	assert.Equal(t, []string{"B", "C", "LP"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A", "C", "LP"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"A", "B", "LP"}, p.CanSwapTo("C"))

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
		})
	}
}

func TestUpdateBalance_Saddle(t *testing.T) {
	t.Parallel()
	// test data from https://etherscan.io/address/0xa6018520eaacc06c30ff2e1b3ee2c7c22e64196a#readContract
	testcases := []struct {
		in               string
		inAmount         string
		out              string
		expectedBalances []string
	}{
		{"A", "10000000", "B",
			[]string{"64752405287165128155", "426593278742291992582", "66589357932477536907", "553429429583268691085"}},
		{"A", "10000000", "C",
			[]string{"64752405287175128155", "426593278742291992582", "66589357932467535940", "553429429583268691085"}},
		{"B", "10000000", "A",
			[]string{"64752405287165221417", "426593278742301992582", "66589357932467535940", "553429429583268691085"}},
		{"C", "9500000000000000000", "B",
			[]string{"64752405287165221417", "417021399572301197888", "76089357932467535940", "553429429583268691085"}},

		// cannot test these case because we haven't accounted for token fee when adding/removing liq yet
		// {"A", 10000000, "LP", []string{"64752405287175220754", "426593278742301992004", "66589357932467535849", "553429429583278677755"}},
		// {"LP", 100000, "A", []string{"64752405287175120660", "426593278742301992004", "66589357932467535849", "553429429583278577755"}},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907",
			"553429429583268691085"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       "{\"initialA\":\"48000\",\"futureA\":\"92000\",\"initialATime\":1652287436,\"futureATime\":1653655053,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\"]}",
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			cloned := p.CloneState()
			amountIn := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
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

			for i, balance := range p.Info.Reserves {
				assert.Equal(t, utils.NewBig10(tc.expectedBalances[i]), balance)
			}
			assert.Equal(t, utils.NewBig10(tc.expectedBalances[len(p.Info.Reserves)]), p.LpSupply)

			clonedRes, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: amountIn,
				TokenOut:      tc.out,
			})
			require.Nil(t, err)
			assert.Equal(t, clonedRes.TokenAmountOut, out.TokenAmountOut)
		})
	}
}
