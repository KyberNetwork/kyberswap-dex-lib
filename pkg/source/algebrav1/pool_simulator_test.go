package algebrav1

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 100, "B", 129970212715135},
		{"A", 10, "B", 11815461610770},
		{"A", 1000, "B", 1310190720959000},
		{"B", 100000000000000000, "A", 75999},
		{"B", 10000000000000000, "A", 7593},
		{"B", 10000000000000000, "A", 7592},
	}
	g := GlobalState{
		Price:              bignumber.NewBig10("90778731131334971326752767343040037"),
		Tick:               big.NewInt(279046),
		Fee:                0,
		TimepointIndex:     62,
		CommunityFeeToken0: 0,
		CommunityFeeToken1: 0,
		Unlocked:           true,
	}
	gs, _ := json.Marshal(g)
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"1156075", "35450062374042037833"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    fmt.Sprintf("{\"tickSpacing\":60,\"liquidity\":119137538372759,\"volumePerLiquidityInBlock\":100,\"totalFeeGrowth0Token\":303633474589870761058966414024, \"totalFeeGrowth1Token\":423897051166683508742054049450199029991046,\"globalState\": %s,\"ticks\":[{\"index\":-887220,\"liquidityGross\":2822091172725,\"liquidityNet\":2822091172725},{\"index\":273540,\"liquidityGross\":116315447200034,\"liquidityNet\":116315447200034},{\"index\":279120,\"liquidityGross\":116315447200034,\"liquidityNet\":-116315447200034},{\"index\":285480,\"liquidityGross\":2822091172725,\"liquidityNet\":-2822091172725},{\"index\":887220,\"liquidityGross\":0,\"liquidityNet\":0}]}", gs),
	}, valueobject.ChainIDPolygon)
	require.Nil(t, err)

	p.feeConf = FeeConfiguration{
		Alpha1:      2900,
		Alpha2:      12000,
		Beta1:       360,
		Beta2:       60000,
		Gamma1:      59,
		Gamma2:      8500,
		VolumeBeta:  0,
		VolumeGamma: 10,
		BaseFee:     100,
	}
	// p.timepoints = TimepointStorage{
	// 	data:    [65536]Timepoint{},
	// 	updates: map[uint16]Timepoint{},
	// }
	// p.timepoints.data[62] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1690340078,
	// 	TickCumulative:                3305222552490,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("34686764562397694336120297099988"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1325981278229"),
	// 	AverageTick:                   278644,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("22914577846679746925"),
	// }
	// p.timepoints.data[61] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1690305248,
	// 	TickCumulative:                3295503380310,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("34584868940404566477249961507713"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1321932456995"),
	// 	AverageTick:                   278368,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("22544148955125044629"),
	// }
	// p.timepoints.data[60] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1690172232,
	// 	TickCumulative:                3258439270006,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("34195728887144793428857869657522"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1313524591763"),
	// 	AverageTick:                   278142,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("22289507597853020474"),
	// }
	// p.timepoints.data[59] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1690067022,
	// 	TickCumulative:                3229152172726,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("33887935651718523488239207364575"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1311733330905"),
	// 	AverageTick:                   278142,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("22081313126730154028"),
	// }
	// p.timepoints.data[56] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1689839577,
	// 	TickCumulative:                3165890225758,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("33222542319986483124737528387109"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1300848732633"),
	// 	AverageTick:                   278160,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("21678436468473874598"),
	// }
	// p.timepoints.data[55] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1689682650,
	// 	TickCumulative:                3122174286733,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("32763450322498238062752310084236"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1291015862858"),
	// 	AverageTick:                   278125,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("21294858696210812038"),
	// }
	// p.timepoints.data[48] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1689246142,
	// 	TickCumulative:                3000877316426,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("31486440441218414479066623931120"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1214431801011"),
	// 	AverageTick:                   279118,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("18729030108416978730"),
	// }
	// p.timepoints.data[47] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1689016063,
	// 	TickCumulative:                2936751308099,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("30813341307926514833903605190516"),
	// 	VolatilityCumulative:          bignumber.NewBig10("1200598010033"),
	// 	AverageTick:                   278676,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("18354898914466789932"),
	// }
	// p.timepoints.data[31] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1683574428,
	// 	TickCumulative:                1423379334857,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("14893766717499460102238987887217"),
	// 	VolatilityCumulative:          bignumber.NewBig10("582317979369"),
	// 	AverageTick:                   275799,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("10357851989805632748"),
	// }
	// p.timepoints.data[32] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1683811741,
	// 	TickCumulative:                1489077065777,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("15588029012706326354667812188387"),
	// 	VolatilityCumulative:          bignumber.NewBig10("647244428973"),
	// 	AverageTick:                   276409,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("10755188147599818688"),
	// }
	// p.timepoints.data[0] = Timepoint{
	// 	Initialized:                   true,
	// 	BlockTimestamp:                1678397959,
	// 	TickCumulative:                0,
	// 	SecondsPerLiquidityCumulative: bignumber.NewBig10("0"),
	// 	VolatilityCumulative:          bignumber.NewBig10("0"),
	// 	AverageTick:                   275768,
	// 	VolumePerLiquidityCumulative:  bignumber.NewBig10("0"),
	// }

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := p.CalcAmountOut(in, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
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
