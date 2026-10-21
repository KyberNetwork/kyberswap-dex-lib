package stable

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

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

// withRamp returns p with A ramping from initialA to futureA between now+startIn and now+endIn.
func withRamp(t *testing.T, p entity.Pool, initialA, futureA string, startIn, endIn int64) entity.Pool {
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	now := time.Now().Unix()
	extra["initialA"], extra["futureA"] = initialA, futureA
	extra["initialATime"], extra["futureATime"] = now+startIn, now+endIn
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	return p
}

// equivalencePools: every fixture, plus A ramping up and down (slowly, so both implementations read
// the same A within a test).
func equivalencePools(t *testing.T) map[string]entity.Pool {
	return map[string]entity.Pool{
		"USD1/USDT":              entityUSD1USDT,
		"USDC/USDT":              entityUSDCUSDT,
		"WBTC/cbBTC":             entityWBTCcbBTC,
		"BSC USDT/USDC":          entityBSCUSDTUSDC,
		"USDC/USDT ramping up":   withRamp(t, entityUSDCUSDT, "472311", "1000000", -1_000_000, 100_000_000),
		"USD1/USDT ramping down": withRamp(t, entityUSD1USDT, "500000", "100000", -1_000_000, 100_000_000),
	}
}

type quote struct {
	amountOut, fee string
	gas            int64
	err            string // "panic: ..." when the legacy implementation panicked
}

func quoteOf(sim poolpkg.IPoolSimulator, in poolpkg.CalcAmountOutParams) (q quote) {
	defer func() {
		if r := recover(); r != nil {
			q = quote{err: fmt.Sprint("panic: ", r)}
		}
	}()
	res, err := sim.CalcAmountOut(in)
	if err != nil {
		return quote{err: err.Error()}
	}
	return quote{amountOut: res.TokenAmountOut.Amount.String(), fee: res.Fee.Amount.String(), gas: res.Gas}
}

// requireSameQuote: the port quotes exactly what the big.Int implementation quoted, fails where it
// failed with the same error, and fails (instead of panicking) where it panicked.
func requireSameQuote(t *testing.T, legacy *legacyPoolSimulator, port *PoolSimulator, in poolpkg.CalcAmountOutParams) quote {
	t.Helper()
	want, got := quoteOf(legacy, in), quoteOf(port, in)
	if len(want.err) > 6 && want.err[:6] == "panic:" {
		require.NotEmpty(t, got.err, "legacy %s, port quoted %+v for %s", want.err, got, in.TokenAmountIn.Amount)
		require.NotContains(t, got.err, "panic")
		return got
	}
	require.Equal(t, want, got, "%s %s→%s", in.TokenAmountIn.Amount, in.TokenAmountIn.Token, in.TokenOut)
	return got
}

// amounts: every power of ten up to 10x the input reserve, and random amounts in between.
func amounts(rng *rand.Rand, reserve *big.Int) []*big.Int {
	var out []*big.Int
	limit := new(big.Int).Mul(reserve, big.NewInt(10))
	for a := big.NewInt(1); a.Cmp(limit) <= 0; a = new(big.Int).Mul(a, big.NewInt(10)) {
		out = append(out, a)
	}
	for range 40 {
		out = append(out, new(big.Int).Rand(rng, new(big.Int).Add(reserve, big.NewInt(1))))
	}
	return out
}

func TestEquivalence_CalcAmountOut(t *testing.T) {
	t.Parallel()
	ok, failed := 0, 0
	for name, p := range equivalencePools(t) {
		legacy, err := newLegacyPoolSimulator(p)
		require.NoError(t, err, name)
		port, err := NewPoolSimulator(p)
		require.NoError(t, err, name)
		rng := rand.New(rand.NewSource(1))
		for i, tokenIn := range port.Info.Tokens {
			for j, tokenOut := range port.Info.Tokens {
				for _, amount := range amounts(rng, port.Info.Reserves[i]) {
					q := requireSameQuote(t, legacy, port, poolpkg.CalcAmountOutParams{
						TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: amount}, TokenOut: tokenOut})
					if q.err == "" {
						ok++
					} else if i != j {
						failed++
					}
				}
			}
		}
	}
	t.Logf("%d quotes succeeded, %d failed, identically", ok, failed)
	require.Greater(t, ok, 200)
	require.Greater(t, failed, 50) // price check, drained side, too-large amounts
}

// Random swaps, applied with UpdateBalance: after every step both implementations hold the same
// reserves and quote the same.
func TestEquivalence_SwapSequences(t *testing.T) {
	t.Parallel()
	swaps := 0
	for name, p := range equivalencePools(t) {
		for seed := range int64(4) {
			legacy, err := newLegacyPoolSimulator(p)
			require.NoError(t, err, name)
			port, err := NewPoolSimulator(p)
			require.NoError(t, err, name)
			rng := rand.New(rand.NewSource(seed))
			for range 150 {
				i := rng.Intn(2)
				tokenIn, tokenOut := port.Info.Tokens[i], port.Info.Tokens[1-i]
				// up to a twentieth of the input reserve
				amount := new(big.Int).Rand(rng, new(big.Int).Add(new(big.Int).Div(port.Info.Reserves[i], big.NewInt(20)), big.NewInt(1)))
				in := poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: amount},
					TokenOut: tokenOut}
				if q := requireSameQuote(t, legacy, port, in); q.err != "" {
					continue
				}
				res, err := legacy.CalcAmountOut(in)
				require.NoError(t, err)
				update := poolpkg.UpdateBalanceParams{TokenAmountIn: in.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
					Fee: *res.Fee, SwapInfo: res.SwapInfo}
				legacy = legacy.CloneState().(*legacyPoolSimulator)
				legacy.UpdateBalance(update)
				port = port.CloneState().(*PoolSimulator)
				port.UpdateBalance(update)
				require.Equal(t, legacy.Info.Reserves, port.Info.Reserves, name)
				swaps++
			}
		}
	}
	t.Logf("%d swaps applied identically", swaps)
	require.Greater(t, swaps, 500)
}

// A ramp that starts in the future reverts on chain (block.timestamp - initial_A_time underflows);
// the big.Int implementation extrapolated A backwards instead.
func TestRampNotStartedFails(t *testing.T) {
	t.Parallel()
	port, err := NewPoolSimulator(withRamp(t, entityUSDCUSDT, "472311", "1000000", 3600, 7200))
	require.NoError(t, err)
	_, err = port.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: port.Info.Tokens[0], Amount: big.NewInt(100e6)}, TokenOut: port.Info.Tokens[1]})
	require.ErrorIs(t, err, errRampNotStarted)
}
