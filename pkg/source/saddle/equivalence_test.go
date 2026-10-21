package saddle

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Ethereum pools of the saddle type, state of 2026-10-10.
var snapshotPools = map[string]string{
	"synapse DAI/USDC/USDT": `{"address":"0x1116898dda4015ed8ddefb84b6e8bc24528af2d8","amplifiedTvl":7798660.829566795,"exchange":"synapse","type":"saddle","timestamp":1791631562,"reserves":["416922515589751765254255","411059435409","1146429084399","1971838973739318893123800"],"tokens":[{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","decimals":6,"swappable":true}],"extra":"{\"initialA\":\"400000\",\"futureA\":\"200000\",\"initialATime\":1653976713,\"futureATime\":1655683200,\"swapFee\":\"1000000\",\"adminFee\":\"10000000000\",\"defaultWithdrawFee\":\"\",\"paused\":false}","staticExtra":"{\"lpToken\":\"0x1b84765de8b7566e4ceaf4d0fd3c5af52d3dde4f\",\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"]}","blockNumber":26161723}`,
	"saddle alETH":          `{"address":"0xa6018520eaacc06c30ff2e1b3ee2c7c22e64196a","amplifiedTvl":260633.07932466798,"exchange":"saddle","type":"saddle","timestamp":1782248772,"reserves":["6586308367942524425","88804198787660388449","6564776058811333730","93533112737500618623"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"swappable":true},{"address":"0x0100546f2cd4c9d97f798ffc9755e47865ff7ee6","decimals":18,"swappable":true},{"address":"0x5e74c9036fb86bd7ecdcb084a0673efc32ea31cb","decimals":18,"swappable":true}],"extra":"{\"initialA\":\"48000\",\"futureA\":\"92000\",\"initialATime\":1652287436,\"futureATime\":1653655053,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\",\"defaultWithdrawFee\":\"\",\"paused\":true}","staticExtra":"{\"lpToken\":\"0xc9da65931abf0ed1b74ce5ad8c041c4220940368\",\"precisionMultipliers\":[\"1\",\"1\",\"1\"]}","blockNumber":25382889}`,
	"saddle 4 tokens":       `{"address":"0xc69ddcd4dfef25d8a793241834d4cc4b3668ead6","reserveUsd":31880.56226390472,"amplifiedTvl":31880.56226390472,"exchange":"saddle","type":"saddle","timestamp":1762910941,"reserves":["10596089074293336370896","5325787915068690526904","4618501953048621873811","9475523362180906458974","29643766959569796847451"],"tokens":[{"address":"0xbc6da0fe9ad5f3b0d58160288917aa56653660e9","decimals":18,"swappable":true},{"address":"0x956f47f50a910163d8bf957cf5846d573e7f87ca","decimals":18,"swappable":true},{"address":"0x853d955acef822db058eb8505911ed77f175b99e","decimals":18,"swappable":true},{"address":"0x5f98805a4e8be255a32880fdec7f6728c6568ba0","decimals":18,"swappable":true}],"extra":"{\"initialA\":\"12000\",\"futureA\":\"24000\",\"initialATime\":1643232168,\"futureATime\":1644666450,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\",\"defaultWithdrawFee\":\"\",\"paused\":true}","staticExtra":"{\"lpToken\":\"0xd48cf4d7fb0824cc8bae055df3092584d0a1726a\",\"precisionMultipliers\":[\"1\",\"1\",\"1\",\"1\"]}"}`,
}

// testPoolABC is TestCalcAmountOut_Saddle's pool (0xa6018520eaacc06c30ff2e1b3ee2c7c22e64196a in 2022).
var testPoolABC = entity.Pool{
	Reserves: entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907",
		"553429429583268691085"},
	Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
	Extra:       `{"initialA":"48000","futureA":"92000","initialATime":1652287436,"futureATime":1653655053,"swapFee":"4000000","adminFee":"5000000000"}`,
	StaticExtra: `{"lpToken":"LP","precisionMultipliers":["1","1","1"]}`,
}

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

// equivalencePools: the snapshot pools, the test pool, and A ramping up and down (slowly, so both
// implementations read the same A within a test).
func equivalencePools(t *testing.T) map[string]entity.Pool {
	pools := map[string]entity.Pool{"test A/B/C": testPoolABC,
		"test A/B/C ramping up":   withRamp(t, testPoolABC, "48000", "92000", -1_000_000, 100_000_000),
		"test A/B/C ramping down": withRamp(t, testPoolABC, "92000", "48000", -1_000_000, 100_000_000)}
	for name, s := range snapshotPools {
		var p entity.Pool
		require.NoError(t, json.Unmarshal([]byte(s), &p))
		pools[name] = p
	}
	return pools
}

type quote struct {
	amountOut, fee string
	gas            int64
	err            string // "panic: ..." when the legacy implementation panicked
}

func quoteOf(sim pool.IPoolSimulator, in pool.CalcAmountOutParams) (q quote) {
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
func requireSameQuote(t *testing.T, legacy *legacyPoolSimulator, port *PoolSimulator, in pool.CalcAmountOutParams) quote {
	t.Helper()
	want, got := quoteOf(legacy, in), quoteOf(port, in)
	if strings.HasPrefix(want.err, "panic:") {
		require.NotEmpty(t, got.err, "legacy %s, port quoted %+v for %s", want.err, got, in.TokenAmountIn.Amount)
		require.NotContains(t, got.err, "panic")
		return got
	}
	require.Equal(t, want, got, "%s %s→%s", in.TokenAmountIn.Amount, in.TokenAmountIn.Token, in.TokenOut)
	return got
}

// amounts: every power of ten up to 10x max, and random amounts up to max.
func amounts(rng *rand.Rand, max *big.Int) []*big.Int {
	var out []*big.Int
	limit := new(big.Int).Mul(max, big.NewInt(10))
	for a := big.NewInt(1); a.Cmp(limit) <= 0; a = new(big.Int).Mul(a, big.NewInt(10)) {
		out = append(out, a)
	}
	for range 25 {
		out = append(out, new(big.Int).Add(new(big.Int).Rand(rng, max), big.NewInt(1)))
	}
	return out
}

// operations lists every (tokenIn, tokenOut, max amount) the pool quotes: swaps, deposits (token → LP)
// and withdrawals (LP → token).
func operations(p *PoolSimulator) (ops [][3]any) {
	for i, tokenIn := range p.Info.Tokens {
		for j, tokenOut := range p.Info.Tokens {
			if i != j {
				ops = append(ops, [3]any{tokenIn, tokenOut, p.Info.Reserves[i]})
			}
		}
		ops = append(ops, [3]any{tokenIn, p.LpToken, p.Info.Reserves[i]})
		ops = append(ops, [3]any{p.LpToken, tokenIn, p.LpSupply.ToBig()})
	}
	return ops
}

func TestEquivalence_CalcAmountOut(t *testing.T) {
	t.Parallel()
	ok, failed := map[string]int{}, 0
	for name, p := range equivalencePools(t) {
		legacy, err := newLegacyPoolSimulator(p)
		require.NoError(t, err, name)
		port, err := NewPoolSimulator(p)
		require.NoError(t, err, name)
		rng := rand.New(rand.NewSource(1))
		for _, op := range operations(port) {
			for _, amount := range amounts(rng, op[2].(*big.Int)) {
				q := requireSameQuote(t, legacy, port, pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: op[0].(string), Amount: amount}, TokenOut: op[1].(string)})
				if q.err != "" {
					failed++
				} else {
					ok[fmt.Sprint(q.gas)]++
				}
			}
		}
	}
	t.Logf("succeeded by gas (swap %d, add %d, remove %d): %v; %d failed, identically", DefaultGas.Swap,
		DefaultGas.AddLiquidity, DefaultGas.RemoveLiquidity, ok, failed)
	for _, gas := range []int64{DefaultGas.Swap, DefaultGas.AddLiquidity, DefaultGas.RemoveLiquidity} {
		require.Greater(t, ok[fmt.Sprint(gas)], 100, "too few successful quotes of gas %d", gas)
	}
	require.Greater(t, failed, 50)
}

// Random swaps, deposits and withdrawals, applied with UpdateBalance: after every step both
// implementations hold the same reserves and LP supply, and quote the same.
func TestEquivalence_Sequences(t *testing.T) {
	t.Parallel()
	applied := 0
	for name, p := range equivalencePools(t) {
		for seed := range int64(3) {
			legacy, err := newLegacyPoolSimulator(p)
			require.NoError(t, err, name)
			port, err := NewPoolSimulator(p)
			require.NoError(t, err, name)
			rng := rand.New(rand.NewSource(seed))
			for range 150 {
				ops := operations(port)
				op := ops[rng.Intn(len(ops))]
				// up to a twentieth of what the input side holds
				max := new(big.Int).Div(op[2].(*big.Int), big.NewInt(20))
				amount := new(big.Int).Add(new(big.Int).Rand(rng, new(big.Int).Add(max, big.NewInt(1))), big.NewInt(1))
				in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: op[0].(string), Amount: amount},
					TokenOut: op[1].(string)}
				if q := requireSameQuote(t, legacy, port, in); q.err != "" {
					continue
				}
				res, err := legacy.CalcAmountOut(in)
				require.NoError(t, err)
				update := pool.UpdateBalanceParams{TokenAmountIn: in.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
					Fee: *res.Fee, SwapInfo: res.SwapInfo}
				legacy = legacy.CloneState().(*legacyPoolSimulator)
				legacy.UpdateBalance(update)
				port = port.CloneState().(*PoolSimulator)
				port.UpdateBalance(update)
				require.Equal(t, legacy.Info.Reserves, port.Info.Reserves, name)
				require.Equal(t, legacy.LpSupply, port.LpSupply.ToBig(), name)
				applied++
			}
		}
	}
	t.Logf("%d operations applied identically", applied)
	require.Greater(t, applied, 500)
}

// The exported big.Int functions wrap the port: where the big.Int implementation returned a positive
// amount they return the same, and where it failed they fail. GetDyUnderlying is checked with and
// without an off-peg fee multiplier.
func TestEquivalence_ExportedFunctions(t *testing.T) {
	t.Parallel()
	same := func(want, wantFee *big.Int, wantErr error, got, gotFee *big.Int, gotErr error) bool {
		if wantErr != nil {
			require.Error(t, gotErr)
			return false
		}
		if want.Sign() <= 0 { // the port returns 0 or an error where big.Int went negative
			return false
		}
		require.NoError(t, gotErr)
		require.Equal(t, want, got)
		require.Equal(t, wantFee, gotFee)
		return true
	}
	checked := 0
	for name, p := range equivalencePools(t) {
		l, err := newLegacyPoolSimulator(p)
		require.NoError(t, err, name)
		rng := rand.New(rand.NewSource(3))
		n := len(l.Info.Tokens)
		for i := range n {
			for _, amount := range amounts(rng, l.Info.Reserves[i]) {
				for j := range n {
					want, wantFee, wantErr := legacyCalculateSwap(l.Info.Reserves, l.Multipliers, l.FutureATime, l.FutureA,
						l.InitialATime, l.InitialA, l.Info.SwapFee, i, j, amount)
					got, gotFee, gotErr := CalculateSwap(l.Info.Reserves, l.Multipliers, l.FutureATime, l.FutureA,
						l.InitialATime, l.InitialA, l.Info.SwapFee, i, j, amount)
					if same(want, wantFee, wantErr, got, gotFee, gotErr) {
						checked++
					}
					for _, feeMul := range []*big.Int{big.NewInt(1e10), big.NewInt(2e10)} {
						want, wantFee, wantErr = legacyGetDyUnderlying(l.Info.Reserves, l.Multipliers, l.FutureATime,
							l.FutureA, l.InitialATime, l.InitialA, l.Info.SwapFee, feeMul, i, j, amount)
						got, gotFee, gotErr = GetDyUnderlying(l.Info.Reserves, l.Multipliers, l.FutureATime, l.FutureA,
							l.InitialATime, l.InitialA, l.Info.SwapFee, feeMul, i, j, amount)
						if same(want, wantFee, wantErr, got, gotFee, gotErr) {
							checked++
						}
					}
				}
				want, wantFee, wantErr := legacyCalculateAddLiquidityOneToken(l.Info.Reserves, l.Multipliers,
					l.FutureATime, l.FutureA, l.InitialATime, l.InitialA, l.DefaultWithdrawFee, l.LpSupply, i, amount)
				got, gotFee, gotErr := CalculateAddLiquidityOneToken(l.Info.Reserves, l.Multipliers, l.FutureATime,
					l.FutureA, l.InitialATime, l.InitialA, l.DefaultWithdrawFee, l.LpSupply, i, amount)
				if same(want, wantFee, wantErr, got, gotFee, gotErr) {
					checked++
				}
			}
			for _, amount := range amounts(rng, l.LpSupply) {
				want, wantFee, wantErr := legacyCalculateRemoveLiquidityOneToken(l.Info.Reserves, l.Multipliers,
					l.FutureATime, l.FutureA, l.InitialATime, l.InitialA, l.Info.SwapFee, l.DefaultWithdrawFee, l.LpSupply,
					i, amount)
				got, gotFee, gotErr := CalculateRemoveLiquidityOneToken(l.Info.Reserves, l.Multipliers, l.FutureATime,
					l.FutureA, l.InitialATime, l.InitialA, l.Info.SwapFee, l.DefaultWithdrawFee, l.LpSupply, i, amount)
				if same(want, wantFee, wantErr, got, gotFee, gotErr) {
					checked++
				}
			}
		}
	}
	t.Logf("%d positive results identical", checked)
	require.Greater(t, checked, 1000)
}

// A ramp that starts in the future reverts on chain (block.timestamp.sub(initialATime) underflows);
// the big.Int implementation extrapolated A backwards instead.
func TestRampNotStartedFails(t *testing.T) {
	t.Parallel()
	port, err := NewPoolSimulator(withRamp(t, testPoolABC, "48000", "92000", 3600, 7200))
	require.NoError(t, err)
	_, err = port.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(1e18)}, TokenOut: "B"})
	require.ErrorIs(t, err, errRampNotStarted)
}

// go test ./pkg/source/saddle/ -run '^$' -bench . -benchmem
func BenchmarkSaddle(b *testing.B) {
	var p entity.Pool
	require.NoError(b, json.Unmarshal([]byte(snapshotPools["synapse DAI/USDC/USDT"]), &p))
	legacy, err := newLegacyPoolSimulator(p)
	require.NoError(b, err)
	port, err := NewPoolSimulator(p)
	require.NoError(b, err)
	dai, usdc, lp := port.Info.Tokens[0], port.Info.Tokens[1], port.LpToken
	for _, c := range []struct {
		name string
		in   pool.CalcAmountOutParams
	}{
		{"swap 1k DAI→USDC", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: dai, Amount: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000))}, TokenOut: usdc}},
		{"add 1k USDC", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(1e9)}, TokenOut: lp}},
		{"remove 1k LP→USDC", pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: lp, Amount: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000))}, TokenOut: usdc}},
	} {
		for name, sim := range map[string]pool.IPoolSimulator{"legacy": legacy, "uint256": port} {
			b.Run(c.name+"/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := sim.CalcAmountOut(c.in); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
