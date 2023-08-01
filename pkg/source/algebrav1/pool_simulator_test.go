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
		{"A", 10, "B", 11734372429489},
		{"B", 100000000000000000, "A", 76522},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"1156075", "35450062374042037833"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    "{\"liquidity\":119137538372759,\"volumePerLiquidityInBlock\":172760224274117266,\"globalState\":{\"price\":90466602735452247634444132580612170,\"tick\":278977,\"fee\":2735,\"timepoint_index\":64,\"community_fee_token0\":0,\"community_fee_token1\":0},\"feeConfig\":{\"alpha1\":2900,\"alpha2\":12000,\"beta1\":360,\"beta2\":60000,\"gamma1\":59,\"gamma2\":8500,\"volumeBeta\":0,\"volumeGamma\":10,\"baseFee\":100},\"ticks\":[{\"Index\":-887220,\"LiquidityGross\":2822091172725,\"LiquidityNet\":2822091172725},{\"Index\":273540,\"LiquidityGross\":116315447200034,\"LiquidityNet\":116315447200034},{\"Index\":279120,\"LiquidityGross\":116315447200034,\"LiquidityNet\":-116315447200034},{\"Index\":285480,\"LiquidityGross\":2822091172725,\"LiquidityNet\":-2822091172725}],\"tickSpacing\":60,\"timepoints\":{\"0\":{\"initialized\":true,\"blockTimestamp\":1678397959,\"tickCumulative\":0,\"secondsPerLiquidityCumulative\":0,\"volatilityCumulative\":0,\"averageTick\":275768,\"volumePerLiquidityCumulative\":0},\"63\":{\"initialized\":true,\"blockTimestamp\":1690436256,\"tickCumulative\":3332060638678,\"secondsPerLiquidityCumulative\":34961469562456729116761598393258,\"volatilityCumulative\":1331162113931,\"averageTick\":279046,\"volumePerLiquidityCumulative\":22914577846679746925},\"64\":{\"initialized\":true,\"blockTimestamp\":1690575362,\"tickCumulative\":3370842000418,\"secondsPerLiquidityCumulative\":35358786139410004212894693879769,\"volatilityCumulative\":1334200963637,\"averageTick\":279046,\"volumePerLiquidityCumulative\":28431625832378378653}}}",
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
		{"B", "100000000000000000", "A", "70214"},
		{"B", "10000000000000000", "A", "6796"},
		{"B", "10000000000000000", "A", "6756"},

		{"A", "1000000000000000000", "B", "35998535759555197554"},
		{"B", "100000", "A", "12291819098246154"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":2822091172725,"volumePerLiquidityInBlock":5957198776710005485,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"fee":1550,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0},"feeConfig":{"alpha1":2900,"alpha2":12000,"beta1":360,"beta2":60000,"gamma1":59,"gamma2":8500,"volumeBeta":0,"volumeGamma":10,"baseFee":100},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60,"timepoints":{"0":{"initialized":true,"blockTimestamp":1678397959,"tickCumulative":0,"secondsPerLiquidityCumulative":0,"volatilityCumulative":0,"averageTick":275768,"volumePerLiquidityCumulative":0},"64":{"initialized":true,"blockTimestamp":1690575362,"tickCumulative":3370842000418,"secondsPerLiquidityCumulative":35358786139410004212894693879769,"volatilityCumulative":1334200963637,"averageTick":279046,"volumePerLiquidityCumulative":28431625832378378653},"65":{"initialized":true,"blockTimestamp":1690856954,"tickCumulative":3449399691802,"secondsPerLiquidityCumulative":36163073298392554504545022580456,"volatilityCumulative":1339141292265,"averageTick":278790,"volumePerLiquidityCumulative":28604386056652495919},"66":{"initialized":false,"blockTimestamp":0,"tickCumulative":0,"secondsPerLiquidityCumulative":0,"volatilityCumulative":0,"averageTick":0,"volumePerLiquidityCumulative":0},"67":{"initialized":false,"blockTimestamp":0,"tickCumulative":0,"secondsPerLiquidityCumulative":0,"volatilityCumulative":0,"averageTick":0,"volumePerLiquidityCumulative":0}}}`,
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
