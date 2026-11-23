package aave

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	// test data from https://etherscan.io/address/0xdebf20617708857ebe4f679508e7b7863a8a8eee#readContract
	// 	call get_dy_underlying to get amount out
	//  (need to be quick because `balances` change rapidly)
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"C", 100000000, "B", 99989535},
		{"C", 1, "A", 999897685887},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"8374598852113385564139023", "8328286891683", "5035549096857"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra: fmt.Sprintf("{\"offpegFeeMultiplier\": \"%v\", \"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"20000000000",
			"4000000",
			"5000000000",
			20000, 200000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"precisionMultipliers\": [\"%v\", \"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\", \"%v\"]}",
			"1", "1000000000000", "1000000000000",
			"Au", "Bu", "Cu"),
	})
	require.Nil(t, err)

	assert.Empty(t, p.CanSwapTo("Cu"))
	assert.Empty(t, p.CanSwapTo("Bu"))
	assert.Empty(t, p.CanSwapTo("Au"))
	assert.Empty(t, p.CanSwapTo("LP"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmtOutResult, error) {
				var in uint256.Int
				in.SetUint64(uint64(tc.inAmount))
				return p.CalcAmtOut(
					pool.CalcAmtOutParams{
						TokenAmountIn: pool.TokenAmt{Token: tc.in, Amount: in},
						TokenOut:      tc.out,
					})
			})
			require.Nil(t, err)
			assert.Equal(t, uint256.NewInt(uint64(tc.expectedOutAmount)).String(), out.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			// no need to check fee, for aave fee has been subtracted from amountOut already
			// if amountOut is correct then so is fee
		})
	}
}

func TestAddLiquidity(t *testing.T) {
	t.Parallel()
	// https://polygonscan.com/address/0x445FE580eF8d70FF569aB36e80c647af338db351#readContract
	testcases := []struct {
		amounts    []string
		expectedLp string
	}{
		{[]string{"10000000", "0", "0"}, "9278148"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"10214770093231964568357824", "7697561755236", "7652555335475", "23723110195594123653004246"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra: fmt.Sprintf("{\"offpegFeeMultiplier\": \"%v\", \"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"20000000000",
			"3000000",
			"5000000000",
			200000, 200000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"precisionMultipliers\": [\"%v\", \"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\", \"%v\"]}",
			"1", "1000000000000", "1000000000000",
			"Au", "Bu", "Cu"),
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			res, err := p.AddLiquidity(lo.Map(tc.amounts, func(s string, _ int) *uint256.Int { return big256.New(s) }))
			require.Nil(t, err)
			assert.Equal(t, big256.New(tc.expectedLp).String(), res.String())
			fmt.Println(p.Info.Reserves)
		})
	}
}

func TestGetDyVirtualPrice(t *testing.T) {
	t.Parallel()
	// https://polygonscan.com/address/0x445FE580eF8d70FF569aB36e80c647af338db351#readContract
	// block 44510081
	testcases := []struct {
		i      int
		j      int
		dx     string
		expOut string
	}{
		{0, 1, "100000000000000", "99"},
		{1, 0, "2", "1999673788966"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"10213317638314302732514558", "7692328822181", "7487545362550", "23563627574547646276749578"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra: fmt.Sprintf("{\"offpegFeeMultiplier\": \"%v\", \"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"20000000000",
			"3000000",
			"5000000000",
			200000, 200000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"precisionMultipliers\": [\"%v\", \"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\", \"%v\"]}",
			"1", "1000000000000", "1000000000000",
			"Au", "Bu", "Cu"),
	})
	require.Nil(t, err)

	v, dCached, err := p.GetVirtualPrice()
	require.Nil(t, err)
	assert.Equal(t, "1077638023314146944", v.String())

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			dy, err := testutil.MustConcurrentSafe(t, func() (*uint256.Int, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, big256.New(tc.dx), nil)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, big256.New(tc.expOut).String(), dy.String())

			// test using cached D
			dy, err = testutil.MustConcurrentSafe(t, func() (*uint256.Int, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, big256.New(tc.dx), dCached)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, big256.New(tc.expOut).String(), dy.String())
		})
	}
}

func TestCloneState(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{"8374598852113385564139023", "8328286891683", "5035549096857"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:    `{"offpegFeeMultiplier":"20000000000","swapFee":"4000000","adminFee":"5000000000","initialA":"20000","futureA":"200000"}`,
		StaticExtra: `{"lpToken":"LP","precisionMultipliers":["1","1000000000000","1000000000000"],` +
			`"underlyingTokens":["Au","Bu","Cu"]}`,
	})
	require.NoError(t, err)

	var in uint256.Int
	in.SetUint64(100000000)
	testutil.TestCloneStateIPoolSim(t, p, pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: "C", Amount: in},
		TokenOut:      "B",
	})
}
