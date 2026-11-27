package smardex

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The Ethereum SDEX/WETH pair 0xf3a4b8ef at block 26375928, as the tracker read it, and a swap of 1 WETH for SDEX
// sent to it on a fork of that block: once as it is, and once after the fork cleared its factory's feeTo
// (0x7753f36e, slot 0). The pair adds the pool fee to feeToAmount only while the factory has a feeTo; without one
// the fee stays in the reserves.
const (
	feeOffPairReserve0 = "44011172171124176810215217"
	feeOffPairReserve1 = "11239751609055145515"
	feeOffPairExtra    = `{"PairFee":{"FeesLP":"5","FeesPool":"2","FeesBase":"10000"},"FictiveReserve":{"FictiveReserve0":"16420840833717183412552335","FictiveReserve1":"5091453093385275044"},"PriceAverage":{"PriceAverage0":"19934279548462101891159222","PriceAverage1":"6255357551948772528","PriceAverageLastTimestamp":"1794209927"},"FeeToAmount":{"Fees0":"11012370104309724727522","Fees1":"2605489557693482"},"FeeOff":true}`
	feeOffSDEX         = "0x5de8ab7e27f6e7a1fff3e5b337584aa43961beef"
	feeOffWETH         = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

func TestPoolFeeFollowsTheFactorysFeeTo(t *testing.T) {
	for _, tc := range []struct {
		name     string
		feeOff   bool
		swapTime int64
		// The pair after the swap: getReserves, getFees.
		reserve1, fees1 string
	}{
		{"feeTo set", false, 1794210416, "12239551609055145515", "2805489557693482"},
		{"no feeTo", true, 1794210403, "12239751609055145515", "2605489557693482"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			atTime(t, tc.swapTime)
			extra := feeOffPairExtra
			if !tc.feeOff {
				// State stored before FeeOff existed reads as fee on.
				extra = strings.Replace(extra, `,"FeeOff":true`, "", 1)
			}
			s, err := NewPoolSimulator(entity.Pool{Address: "0xf3a4b8efe3e3049f6bc71b47ccb7ce6665420179",
				Reserves: entity.PoolReserves{feeOffPairReserve0, feeOffPairReserve1},
				Tokens:   []*entity.PoolToken{{Address: feeOffSDEX}, {Address: feeOffWETH}}, Extra: extra})
			require.NoError(t, err)

			res, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: feeOffWETH, Amount: big.NewInt(1e18)}, TokenOut: feeOffSDEX})
			require.NoError(t, err)
			require.Equal(t, "2776892190115132227287221", res.TokenAmountOut.Amount.String())
			s.UpdateBalance(poolpkg.UpdateBalanceParams{SwapInfo: res.SwapInfo})

			require.Equal(t, []string{"41234279981009044582927996", tc.reserve1},
				[]string{s.Info.Reserves[0].String(), s.Info.Reserves[1].String()})
			require.Equal(t, []string{"11012370104309724727522", tc.fees1},
				[]string{s.FeeToAmount.Fees0.Dec(), s.FeeToAmount.Fees1.Dec()})
			require.Equal(t, []string{"17288449834144118319135128", "7221268727075042076"},
				[]string{s.FictiveReserve.FictiveReserve0.Dec(), s.FictiveReserve.FictiveReserve1.Dec()})
		})
	}
}
