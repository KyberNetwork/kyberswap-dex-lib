package arberaden

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x9d8890bb264de97bba37f4a512b2fc2fa08d06f0","exchange":"arbera-den","type":"arbera-den","timestamp":1759290395,"tokens":[{"address":"0x9d8890bb264de97bba37f4a512b2fc2fa08d06f0","symbol":"brNECT","decimals":18,"swappable":true},{"address":"0x1ce0a25d13ce4d52071ae7e02cf1f6606f4c79d3","symbol":"NECT","decimals":18,"swappable":true}],"extra":"{\"assets\":[{\"token\":\"0x1ce0a25d13ce4d52071ae7e02cf1f6606f4c79d3\",\"weighting\":\"1000000000000000000\",\"basePriceUSDX96\":\"0\",\"c1\":\"0x0000000000000000000000000000000000000000\",\"q1\":\"79228162514264337593543950336000000000000000000\"}],\"assetSupplies\":[\"177765524182485275394558\"],\"supply\":\"174697524527989303631620\",\"fee\":{\"bond\":\"20\",\"debond\":\"20\",\"burn\":\"7000\"}}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

// TestPoolSimulator_CalcAmtOutGolden pins the native quote to the on-chain simulations linked below.
func TestPoolSimulator_CalcAmtOutGolden(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		tokenIn  int
		amountIn string
		wantOut  string
	}{
		// https://www.tdly.co/shared/simulation/55c5b682-43bb-4d91-8243-e37afe5f220c
		{"bond-1e18", 1, "1000000000000000000", "980775829738257783"},
		// https://www.tdly.co/shared/simulation/fa0f7349-e2e2-4095-ad56-89becb2241d2
		{"debond-1e18", 0, "1000000000000000000", "1015526657366552557"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: poolSim.Info.Tokens[tc.tokenIn], Amount: *uint256.MustFromDecimal(tc.amountIn)},
				TokenOut:      poolSim.Info.Tokens[1-tc.tokenIn],
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantOut, res.TokenAmountOut.Amount.Dec())
		})
	}
}

// TestPoolSimulator_CalcAmtOutWithUpdateState replays the swap sequence on a clone, checking each output.
func TestPoolSimulator_CalcAmtOutWithUpdateState(t *testing.T) {
	cases := []struct {
		name    string
		tokenIn int
		steps   [][2]string
	}{
		{"bond", 1, [][2]string{
			{"1000000000000000000", "980775829738257783"},
			{"1000000000000000000", "980786856564798846"},
		}},
		{"debond", 0, [][2]string{
			{"1000000000000000000", "1015526657366552557"},
			{"1000000000000000000", "1015526665504878830"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sim := poolSim.CloneState().(*PoolSimulator)
			tokenIn, tokenOut := sim.Info.Tokens[tc.tokenIn], sim.Info.Tokens[1-tc.tokenIn]
			for i, step := range tc.steps {
				amtIn := uint256.MustFromDecimal(step[0])
				res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtIn},
					TokenOut:      tokenOut,
				})
				require.NoError(t, err, "step %d", i)
				require.Equal(t, step[1], res.TokenAmountOut.Amount.Dec(), "step %d", i)
				sim.UpdateState(pool.UpdateStateParams{
					TokenAmountIn:  pool.TokenAmt{Token: tokenIn, Amount: *amtIn},
					TokenAmountOut: *res.TokenAmountOut,
					SwapInfo:       res.SwapInfo,
				})
			}
		})
	}
}
