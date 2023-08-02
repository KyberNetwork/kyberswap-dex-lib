package algebrav1

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"fee":2985,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	}, valueobject.ChainIDPolygon)
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
	logger.SetLogLevel("debug")
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
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"fee":2979,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	}, valueobject.ChainIDPolygon)
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

func TestPoolSimulator_TTT(t *testing.T) {
	logger.SetLogLevel("debug")
	// s := ``
	ts := TimepointStorage{
		data: map[uint16]Timepoint{
			3761: {
				Initialized:                   true,
				BlockTimestamp:                1690886470,
				TickCumulative:                -7923821458525,
				SecondsPerLiquidityCumulative: bignumber.NewBig10("19039813338409223274655056155053"),
				VolatilityCumulative:          bignumber.NewBig10("2050833005535"),
				AverageTick:                   -280087,
				VolumePerLiquidityCumulative:  bignumber.NewBig10("1999071478699648010695"),
			},
			3762: {
				Initialized:                   true,
				BlockTimestamp:                1689103076,
				TickCumulative:                -7425818620719,
				SecondsPerLiquidityCumulative: bignumber.NewBig10("19039777215785617957477406702521"),
				VolatilityCumulative:          bignumber.NewBig10("1949636424824"),
				AverageTick:                   -279339,
				VolumePerLiquidityCumulative:  bignumber.NewBig10("1877927962015256631686"),
			},
		},
		updates: map[uint16]Timepoint{},
	}
	// json.Unmarshal([]byte(s), &ts.data)

	now := uint32(1690887021)
	err, b, e := ts.binarySearch(now, now-3600*24, 3761, 3762)
	fmt.Println(err, b, e)
}
