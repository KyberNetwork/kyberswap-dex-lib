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
		{"A", 10, "B", 11815484110223},
		{"B", 100000000000000000, "A", 75997},
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
		Extra:    fmt.Sprintf("{\"liquidity\":119137538372759,\"volumePerLiquidityInBlock\":100,\"totalFeeGrowth0Token\":303633474589870761058966414024, \"totalFeeGrowth1Token\":423897051166683508742054049450199029991046,\"globalState\": %s,\"ticks\":[{\"index\":-887220,\"liquidityGross\":2822091172725,\"liquidityNet\":2822091172725},{\"index\":273540,\"liquidityGross\":116315447200034,\"liquidityNet\":116315447200034},{\"index\":279120,\"liquidityGross\":116315447200034,\"liquidityNet\":-116315447200034},{\"index\":285480,\"liquidityGross\":2822091172725,\"liquidityNet\":-2822091172725},{\"index\":887220,\"liquidityGross\":0,\"liquidityNet\":0}]}", gs),
	}, valueobject.ChainIDPolygon)
	require.Nil(t, err)

	p.feeConf = FeeConfiguration{
		alpha1:      2900,
		alpha2:      12000,
		beta1:       360,
		beta2:       60000,
		gamma1:      59,
		gamma2:      8500,
		volumeBeta:  0,
		volumeGamma: 10,
		baseFee:     100,
	}
	p.timepoints = TimepointStorage{
		data:    [65536]Timepoint{},
		updates: map[uint16]Timepoint{},
	}
	p.timepoints.data[62] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1690340078,
		tickCumulative:                3305222552490,
		secondsPerLiquidityCumulative: bignumber.NewBig10("34686764562397694336120297099988"),
		volatilityCumulative:          bignumber.NewBig10("1325981278229"),
		averageTick:                   278644,
		volumePerLiquidityCumulative:  bignumber.NewBig10("22914577846679746925"),
	}
	p.timepoints.data[61] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1690305248,
		tickCumulative:                3295503380310,
		secondsPerLiquidityCumulative: bignumber.NewBig10("34584868940404566477249961507713"),
		volatilityCumulative:          bignumber.NewBig10("1321932456995"),
		averageTick:                   278368,
		volumePerLiquidityCumulative:  bignumber.NewBig10("22544148955125044629"),
	}
	p.timepoints.data[60] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1690172232,
		tickCumulative:                3258439270006,
		secondsPerLiquidityCumulative: bignumber.NewBig10("34195728887144793428857869657522"),
		volatilityCumulative:          bignumber.NewBig10("1313524591763"),
		averageTick:                   278142,
		volumePerLiquidityCumulative:  bignumber.NewBig10("22289507597853020474"),
	}
	p.timepoints.data[59] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1690067022,
		tickCumulative:                3229152172726,
		secondsPerLiquidityCumulative: bignumber.NewBig10("33887935651718523488239207364575"),
		volatilityCumulative:          bignumber.NewBig10("1311733330905"),
		averageTick:                   278142,
		volumePerLiquidityCumulative:  bignumber.NewBig10("22081313126730154028"),
	}
	p.timepoints.data[56] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1689839577,
		tickCumulative:                3165890225758,
		secondsPerLiquidityCumulative: bignumber.NewBig10("33222542319986483124737528387109"),
		volatilityCumulative:          bignumber.NewBig10("1300848732633"),
		averageTick:                   278160,
		volumePerLiquidityCumulative:  bignumber.NewBig10("21678436468473874598"),
	}
	p.timepoints.data[55] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1689682650,
		tickCumulative:                3122174286733,
		secondsPerLiquidityCumulative: bignumber.NewBig10("32763450322498238062752310084236"),
		volatilityCumulative:          bignumber.NewBig10("1291015862858"),
		averageTick:                   278125,
		volumePerLiquidityCumulative:  bignumber.NewBig10("21294858696210812038"),
	}
	p.timepoints.data[48] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1689246142,
		tickCumulative:                3000877316426,
		secondsPerLiquidityCumulative: bignumber.NewBig10("31486440441218414479066623931120"),
		volatilityCumulative:          bignumber.NewBig10("1214431801011"),
		averageTick:                   279118,
		volumePerLiquidityCumulative:  bignumber.NewBig10("18729030108416978730"),
	}
	p.timepoints.data[47] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1689016063,
		tickCumulative:                2936751308099,
		secondsPerLiquidityCumulative: bignumber.NewBig10("30813341307926514833903605190516"),
		volatilityCumulative:          bignumber.NewBig10("1200598010033"),
		averageTick:                   278676,
		volumePerLiquidityCumulative:  bignumber.NewBig10("18354898914466789932"),
	}
	p.timepoints.data[31] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1683574428,
		tickCumulative:                1423379334857,
		secondsPerLiquidityCumulative: bignumber.NewBig10("14893766717499460102238987887217"),
		volatilityCumulative:          bignumber.NewBig10("582317979369"),
		averageTick:                   275799,
		volumePerLiquidityCumulative:  bignumber.NewBig10("10357851989805632748"),
	}
	p.timepoints.data[32] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1683811741,
		tickCumulative:                1489077065777,
		secondsPerLiquidityCumulative: bignumber.NewBig10("15588029012706326354667812188387"),
		volatilityCumulative:          bignumber.NewBig10("647244428973"),
		averageTick:                   276409,
		volumePerLiquidityCumulative:  bignumber.NewBig10("10755188147599818688"),
	}
	p.timepoints.data[0] = Timepoint{
		initialized:                   true,
		blockTimestamp:                1678397959,
		tickCumulative:                0,
		secondsPerLiquidityCumulative: bignumber.NewBig10("0"),
		volatilityCumulative:          bignumber.NewBig10("0"),
		averageTick:                   275768,
		volumePerLiquidityCumulative:  bignumber.NewBig10("0"),
	}

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}
