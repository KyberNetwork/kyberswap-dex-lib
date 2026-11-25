package stable

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// replayNow pins the ramp clock. Every ramp in testdata/legacy_quotes.json ends at replayNow, so A reads futureA
// on every run and the recorded quotes do not drift with time.Now().
const replayNow int64 = 1_780_000_000

type goldenCase struct {
	Pool      string `json:"pool"`
	Kind      string `json:"kind"`
	Seed      int64  `json:"seed"`
	Step      int    `json:"step"`
	Direction string `json:"direction"`
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	Amount    string `json:"amount"`
	AmountOut string `json:"amountOut"`
	Fee       string `json:"fee"`
	Gas       int64  `json:"gas"`
	Err       string `json:"err"`
}

func replayRamp(t *testing.T, p entity.Pool, initialA, futureA string, startIn, endIn int64) entity.Pool {
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra["initialA"], extra["futureA"] = initialA, futureA
	extra["initialATime"], extra["futureATime"] = replayNow+startIn, replayNow+endIn
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	return p
}

func replayPools(t *testing.T) map[string]entity.Pool {
	return map[string]entity.Pool{
		"USD1/USDT":              entityUSD1USDT,
		"USDC/USDT":              entityUSDCUSDT,
		"WBTC/cbBTC":             entityWBTCcbBTC,
		"BSC USDT/USDC":          entityBSCUSDTUSDC,
		"USDC/USDT ramping up":   replayRamp(t, entityUSDCUSDT, "472311", "1000000", -1_000_000, 0),
		"USD1/USDT ramping down": replayRamp(t, entityUSD1USDT, "500000", "100000", -1_000_000, 0),
	}
}

// entityBSCUSDTUSDC is TestCloneState's pool: USDT/USDC on BSC, both 18 decimals, ~6e7 of each.
var entityBSCUSDTUSDC = entity.Pool{
	Address:  "0xf5448fc2beb9324900d08225fe4530ba3bbf654f",
	Exchange: "lista-stable",
	Type:     "lista-stable",
	Reserves: entity.PoolReserves{"48615751411650460241085692", "11206579925899312237017692", "59769030327001165128372730"},
	Tokens: []*entity.PoolToken{
		{Address: "0x55d398326f99059ff775485246999027b3197955", Symbol: "USDT", Decimals: 18},
		{Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Symbol: "USDC", Decimals: 18},
	},
	Extra:       `{"initialA":"500000","futureA":"500000","initialATime":0,"futureATime":0,"swapFee":"100000","adminFee":"2000000000","oraclePrices":[998996790000000000,999662870000000000],"priceDiffThreshold":[50000000000000000,50000000000000000]}`,
	StaticExtra: `{"lpToken":"0xF6136d7e72446C724ecAeef514AE7B2ab4dbb60B","aPrecision":"100","precisionMultipliers":["1","1"],"rates":["1000000000000000000","1000000000000000000"],"isNativeCoins":[false,false]}`,
}

func parseAmount(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok, s)
	return v
}

func replayQuote(t *testing.T, sim poolpkg.IPoolSim, c goldenCase) goldenCase {
	t.Helper()
	res, err := sim.CalcAmtOut(poolpkg.CalcAmtOutParams{
		TokenAmountIn: poolpkg.TokenAmt{Token: c.TokenIn, Amount: *uint256.MustFromBig(parseAmount(t, c.Amount))},
		TokenOut:      c.TokenOut,
	})
	if err != nil {
		return goldenCase{Err: err.Error()}
	}
	return goldenCase{AmountOut: res.TokenAmountOut.Amount.Dec(), Gas: res.Gas}
}

// TestReplayLegacyQuotes: the production simulator, built through FactoryPoolSim, quotes every recorded case the
// same as the big.Int implementation before the uint256 port, sweep and swap sequences alike.
func TestReplayLegacyQuotes(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/legacy_quotes.json")
	require.NoError(t, err)
	var cases []goldenCase
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases)
	for i := range cases { // the fee is no longer on the native result
		cases[i].Fee = ""
	}

	pools := replayPools(t)
	factory := poolpkg.FactoryPoolSim(DexType)
	require.NotNil(t, factory)
	newSim := func(name string) poolpkg.IPoolSim {
		sim, err := factory(poolpkg.FactoryParams{EntityPool: pools[name]})
		require.NoError(t, err, name)
		return sim
	}

	sweep, swaps := 0, 0
	sims := map[string]poolpkg.IPoolSim{}
	for _, c := range cases {
		switch c.Kind {
		case "sweep":
			got := replayQuote(t, newSim(c.Pool), c)
			got.Pool, got.Kind, got.Seed, got.Direction, got.TokenIn, got.TokenOut, got.Amount = c.Pool, c.Kind, c.Seed, c.Direction, c.TokenIn, c.TokenOut, c.Amount
			require.Equal(t, c, got, "%s %s %s→%s", c.Pool, c.Amount, c.TokenIn, c.TokenOut)
			sweep++
		case "swap":
			key := c.Pool + "#" + big.NewInt(c.Seed).String()
			sim, ok := sims[key]
			if !ok {
				sim = newSim(c.Pool)
				sims[key] = sim
			}
			got := replayQuote(t, sim, c)
			got.Pool, got.Kind, got.Seed, got.Step, got.Direction, got.TokenIn, got.TokenOut, got.Amount = c.Pool, c.Kind, c.Seed, c.Step, c.Direction, c.TokenIn, c.TokenOut, c.Amount
			require.Equal(t, c, got, "%s seed %d step %d", c.Pool, c.Seed, c.Step)
			if c.Err != "" {
				continue
			}
			sim.UpdateState(poolpkg.UpdateStateParams{
				TokenAmountIn:  poolpkg.TokenAmt{Token: c.TokenIn, Amount: *uint256.MustFromBig(parseAmount(t, c.Amount))},
				TokenAmountOut: poolpkg.TokenAmt{Token: c.TokenOut, Amount: *uint256.MustFromBig(parseAmount(t, c.AmountOut))},
			})
			swaps++
		default:
			t.Fatalf("unknown kind %q", c.Kind)
		}
	}
	t.Logf("%d sweep quotes and %d applied swaps, identical", sweep, swaps)
}
