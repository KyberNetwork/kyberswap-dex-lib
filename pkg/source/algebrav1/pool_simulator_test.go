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

// TODO: test pool with community_fee_token0 https://bscscan.com/address/0x0137a5ba1dfa5d6d9a5896251f3d06b2e6669c3a#readContract
