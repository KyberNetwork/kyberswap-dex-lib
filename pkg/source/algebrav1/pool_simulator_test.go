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
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2985,"feeOtz":2985,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
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
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2979,"feeOtz":2979,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
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
		Extra:    `{"liquidity":0,"globalState":{"price":4295128740,"tick":-887272,"feeZto":1622,"feeOtz":1622,"timepoint_index":2497,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-3420,"LiquidityGross":3425867281055637406,"LiquidityNet":3425867281055637406},{"Index":-1680,"LiquidityGross":54492387444405553633,"LiquidityNet":54492387444405553633},{"Index":-1500,"LiquidityGross":11191922902152224210,"LiquidityNet":11191922902152224210},{"Index":0,"LiquidityGross":2148740956490219135,"LiquidityNet":2148740956490219135},{"Index":60,"LiquidityGross":5964987541425314734,"LiquidityNet":5964987541425314734},{"Index":120,"LiquidityGross":5964987541425314734,"LiquidityNet":-5964987541425314734},{"Index":180,"LiquidityGross":2148740956490219135,"LiquidityNet":-2148740956490219135},{"Index":1200,"LiquidityGross":54492387444405553633,"LiquidityNet":-54492387444405553633},{"Index":1380,"LiquidityGross":11191922902152224210,"LiquidityNet":-11191922902152224210},{"Index":2160,"LiquidityGross":3425867281055637406,"LiquidityNet":-3425867281055637406}],"tickSpacing":60}`,
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
		Extra:    `{"liquidity":98714460437307995596273,"globalState":{"price":1572768200222810245774927517376,"tick":59768,"feeZto":11076,"feeOtz":11076,"timepoint_index":45,"community_fee_token0":1000,"community_fee_token1":1000,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":98714460437307995596273},{"Index":887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":-98714460437307995596273}],"tickSpacing":60}`,
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

func TestPoolSimulator_CalcAmountOut_v1_9(t *testing.T) {
	// test data from https://ftmscan.com/address/0x2fbb6b6c054ef35f20c91fd29d6579cb3c642195#code
	// tickSpacing=5
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "3"},
		{"A", "10000000000", "B", "4041064818"},
		{"B", "10000000000", "A", "24373699676"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"21265875874493991905878", "10344609910613908943698"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":299344339249801237803452,"globalState":{"price":50556054571765543459252266509,"tick":-8986,"feeZto":7550,"feeOtz":7550,"timepoint_index":4,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-23040,"LiquidityGross":18101291400643986804037,"LiquidityNet":18101291400643986804037},{"Index":-9495,"LiquidityGross":281243047849157250999415,"LiquidityNet":281243047849157250999415},{"Index":-8940,"LiquidityGross":281243047849157250999415,"LiquidityNet":-281243047849157250999415},{"Index":16080,"LiquidityGross":18101291400643986804037,"LiquidityNet":-18101291400643986804037}],"tickSpacing":5}`,
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

func TestPoolSimulator_CalcAmountOut_DirFee(t *testing.T) {
	// test data from https://arbiscan.io/address/0x2f0bcb4a8bd714953eefd5339326ee0ff62c5b62#readContract
	// different fee for 2 directions
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10000000000, "B", 11273265321},
		{"B", 10000000000, "A", 8843048322},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
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

func TestPoolSimulator_UpdateBalance_DirFee(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://arbiscan.io/address/0x2f0bcb4a8bd714953eefd5339326ee0ff62c5b62#readContract
	// different fee for 2 directions
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "10"},
		{"A", "100", "B", "111"},
		{"A", "1000", "B", "1126"},
		{"B", "10000000000000", "A", "8843048235909"},
		{"B", "100000000000", "A", "88430481480"},
		{"B", "100000000000", "A", "88430481462"},

		{"A", "100000000000000", "B", "112732642939385"},
		{"B", "100000", "A", "88430"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
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
