package algebrav1

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10, "B", 12418116005823},
		{"B", 100000000000000000, "A", 70148},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"fee":2985,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	}, 1001)
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := p.CalcAmountOut(in, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "12418116005823"},
		{"A", "100", "B", "136593135772329"},
		{"A", "1000", "B", "1374962214882655"},
		{"B", "100000000000000000", "A", "70212"},
		{"B", "10000000000000000", "A", "6796"},
		{"B", "10000000000000000", "A", "6756"},

		{"A", "1000000000000000000", "B", "35998532399555197330"},
		{"B", "100000", "A", "12290889471038619"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"fee":2979,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	}, 1001)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(in, tc.out)
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})
		})
	}
}

func TestPoolSimulator_CalcAmountOut_SPL(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://polygonscan.com/address/0x63aefd3aefeedce0860a5ef21c1af548641620dd#readContract
	testcases := []struct {
		in       string
		inAmount int64
		out      string
	}{
		{"A", 10, "B"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"10963601168695220226", "357336560175387760"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":0,"globalState":{"price":4295128740,"tick":-887272,"fee":1622,"timepoint_index":2497,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-3420,"LiquidityGross":3425867281055637406,"LiquidityNet":3425867281055637406},{"Index":-1680,"LiquidityGross":54492387444405553633,"LiquidityNet":54492387444405553633},{"Index":-1500,"LiquidityGross":11191922902152224210,"LiquidityNet":11191922902152224210},{"Index":0,"LiquidityGross":2148740956490219135,"LiquidityNet":2148740956490219135},{"Index":60,"LiquidityGross":5964987541425314734,"LiquidityNet":5964987541425314734},{"Index":120,"LiquidityGross":5964987541425314734,"LiquidityNet":-5964987541425314734},{"Index":180,"LiquidityGross":2148740956490219135,"LiquidityNet":-2148740956490219135},{"Index":1200,"LiquidityGross":54492387444405553633,"LiquidityNet":-54492387444405553633},{"Index":1380,"LiquidityGross":11191922902152224210,"LiquidityNet":-11191922902152224210},{"Index":2160,"LiquidityGross":3425867281055637406,"LiquidityNet":-3425867281055637406}],"tickSpacing":60}`,
	}, 1001)
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			_, err := p.CalcAmountOut(in, tc.out)
			require.NotNil(t, err)
			assert.Contains(t, err.Error(), ErrSPL.Error())
		})
	}
}

func TestPoolSimulator_CalcAmountOut_CommFee(t *testing.T) {
	// test data from https://bscscan.com/address/0x0137a5ba1dfa5d6d9a5896251f3d06b2e6669c3a#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "3546"},
		{"A", "100", "B", "38618"},
		{"A", "1000", "B", "389338"},
		{"B", "100000000000000000", "A", "250953133732636"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"4972738711862929441043", "1959593146565760679885786"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":98714460437307995596273,"globalState":{"price":1572768200222810245774927517376,"tick":59768,"fee":11076,"timepoint_index":45,"community_fee_token0":1000,"community_fee_token1":1000,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":98714460437307995596273},{"Index":887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":-98714460437307995596273}],"tickSpacing":60}`,
	}, 1001)
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(in, tc.out)
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
