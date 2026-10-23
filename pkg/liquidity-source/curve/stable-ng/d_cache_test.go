package stableng

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// crvUSD/USDC.e (Arbitrum, from pool_simulator_test.go) and two 3-coin pools.
var dCacheTestPools = []string{
	`{"address":"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709287180,"reserves":["8994725349517509957774712","1568153728639","10550045569550900254909685"],"tokens":[{"address":"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"1000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"50000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\"}","blockNumber":185977087}`,
	// Ethereum, state of 2026-10-10: sUSDe/sDAI/sFRAX (vault rates) and EURC/EURS/EURQ (6, 2 and 6 decimals).
	`{"address":"0x48edff046ebf41c6ee1895e06f3dc1629db563d4","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1791532572,"reserves":["894557164","93357","9171880210","10734141710720494092360"],"tokens":[{"address":"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c","decimals":6,"swappable":true},{"address":"0xdb25f211ab05b1c97d595516f45794528a807ad8","decimals":2,"swappable":true},{"address":"0x8df723295214ea6f21026eeeb4382d475f146f9f","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"100000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"10000000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"100000000000\",\"IsNativeCoins\":[false,false,false]}","blockNumber":26153495}`,
	`{"address":"0xc559f6716d8b1471fc2dc10aafeb0faa219fe9df","amplifiedTvl":2055.0658573472438,"exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1791633588,"reserves":["30493870773404389395","32589989272256370584","202550469831335536561","242449429390699592715"],"tokens":[{"address":"0x9d39a5de30e57443bff2a8307a4256c8797a3497","decimals":18,"swappable":true},{"address":"0x83f20f44975d03b1b09e64809b757c47f942beea","decimals":18,"swappable":true},{"address":"0xa663b02cf0a4b149d2ad41910cb81e23e1c41c32","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"100000\",\"FutureA\":\"100000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"2000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"50000000000\",\"RateMultipliers\":[\"1252345287555579438\",\"1181562001899517117\",\"1160779691465741700\"]}","staticExtra":"{\"APrecision\":\"100\",\"OffpegFeeMultiplier\":\"50000000000\",\"IsNativeCoins\":[false,false,false]}","blockNumber":26161890}`,
}

func newDCacheTestPool(t testing.TB, poolJSON string, edit func(*entity.Pool)) *PoolSimulator {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolJSON), &p))
	if edit != nil {
		edit(&p)
	}
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

// uncached is a copy of s that solves for D on every quote, as before the cache.
func uncached(s *PoolSimulator) *PoolSimulator {
	c := s.CloneState().(*PoolSimulator)
	c.dCache.ok = false
	return c
}

// requireSameQuotes quotes every pair of s, both ways, at amounts from dust to a fifth of the reserve,
// and requires exactly what the uncached copy quotes: amount, fee and error.
func requireSameQuotes(t *testing.T, s *PoolSimulator, rng *rand.Rand) {
	t.Helper()
	ref := uncached(s)
	for i, tokenIn := range s.Info.Tokens {
		for j, tokenOut := range s.Info.Tokens {
			if i == j {
				continue
			}
			for range 4 {
				amount := new(big.Int).Div(s.Info.Reserves[i], big.NewInt(int64(5+rng.IntN(1_000_000))))
				amount.Add(amount, big.NewInt(1))
				in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount},
					TokenOut: tokenOut}
				got, gotErr := s.CalcAmountOut(in)
				want, wantErr := ref.CalcAmountOut(in)
				require.Equal(t, errMsg(wantErr), errMsg(gotErr))
				if wantErr == nil {
					require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount, "%s→%s %s", tokenIn, tokenOut, amount)
					require.Equal(t, want.Fee.Amount, got.Fee.Amount)
				}

				out := pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: amount},
					TokenIn: tokenIn}
				gotIn, gotErr := s.CalcAmountIn(out)
				wantIn, wantErr := ref.CalcAmountIn(out)
				require.Equal(t, errMsg(wantErr), errMsg(gotErr))
				if wantErr == nil {
					require.Equal(t, wantIn.TokenAmountIn.Amount, gotIn.TokenAmountIn.Amount)
					require.Equal(t, wantIn.Fee.Amount, gotIn.Fee.Amount)
				}
			}
		}
	}
}

func requireCacheMiss(t *testing.T, s *PoolSimulator) {
	t.Helper()
	d, err := s.cachedD(XpMem(s.Extra.RateMultipliers, s.Reserves))
	require.Nil(t, d)
	require.NoError(t, err)
}

// errMsg compares errors by message: wrapped errors carry the stack they were raised from.
func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// The cache must not change any quote, and it must stay warm through swaps: a cold cache is correct
// but slow, which is what the cache is for.
func TestDCacheMatchesUncachedThroughSwaps(t *testing.T) {
	t.Parallel()
	for _, poolJSON := range dCacheTestPools {
		s := newDCacheTestPool(t, poolJSON, nil)
		rng := rand.New(rand.NewPCG(1, 2))
		for step := range 50 {
			d, err := s.cachedD(XpMem(s.Extra.RateMultipliers, s.Reserves))
			require.NoError(t, err)
			require.NotNil(t, d, "step %d: cold cache", step)
			requireSameQuotes(t, s, rng)

			i := rng.IntN(len(s.Info.Tokens))
			j := (i + 1 + rng.IntN(len(s.Info.Tokens)-1)) % len(s.Info.Tokens)
			amountIn := pool.TokenAmount{Token: s.Info.Tokens[i],
				Amount: new(big.Int).Div(s.Info.Reserves[i], big.NewInt(int64(20+rng.IntN(1000))))}
			res, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: amountIn, TokenOut: s.Info.Tokens[j]})
			if err != nil {
				continue
			}
			s = s.CloneState().(*PoolSimulator)
			s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: amountIn, TokenAmountOut: *res.TokenAmountOut,
				Fee: *res.Fee, SwapInfo: res.SwapInfo})
		}
	}
}

// A pool whose D does not converge (Test_PanicCase's pool) caches the failure, and fails every quote
// with the error it failed with before, including the checks GetY makes before solving for D.
func TestDCacheKeepsDFailure(t *testing.T) {
	t.Parallel()
	s := newDCacheTestPool(t, `{"address":"0x126331661101057c3a5879ac6af3f30cac6c66e2","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1755684283,"reserves":["119939676640073155282621801","29312707","26131114","29544061845500753036","16147198280723800","17557865949750060591","141230171454148761985"],"tokens":[{"address":"0x5905c0290e59afbdc42a5480add9bd7b20cb3759","symbol":"BLX","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x6b175474e89094c44da98b954eedeac495271d0f","symbol":"DAI","decimals":18,"swappable":true},{"address":"0x0000000000085d4780b73119b644ae5ecd22b376","symbol":"TUSD","decimals":18,"swappable":true},{"address":"0x4fabb145d64652a948d72533023f6e7a623c7c53","symbol":"BUSD","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"20000000000\",\"RateMultipliers\":[\"1000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000\",\"1000000000000000000\",\"1000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\",\"IsNativeCoins\":[false,false,false,false,false,false]}","blockNumber":23181576}`, nil)
	require.True(t, s.dCache.ok)
	require.ErrorIs(t, s.dCache.err, ErrDDoesNotConverge)
	for _, token := range s.Info.Tokens { // same token in and out: ErrTokenFromEqualsTokenTo, not D's error
		in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: token, Amount: big.NewInt(1e6)},
			TokenOut: token}
		_, gotErr := s.CalcAmountOut(in)
		_, wantErr := uncached(s).CalcAmountOut(in)
		require.Error(t, wantErr)
		require.Equal(t, errMsg(wantErr), errMsg(gotErr))
	}
	requireSameQuotes(t, s, rand.New(rand.NewPCG(7, 8)))
}

// Quotes must take D, or the error solving for it, from the cache: that is the whole saving, and an
// unused cache would still pass every equivalence test. A planted D and a planted error show through.
func TestDCacheIsUsed(t *testing.T) {
	t.Parallel()
	s := newDCacheTestPool(t, dCacheTestPools[0], nil)
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0], Amount: big.NewInt(1e18)},
		TokenOut: s.Info.Tokens[1]}
	want, err := s.CalcAmountOut(in)
	require.NoError(t, err)
	s.dCache.d.Add(&s.dCache.d, uint256.NewInt(1e15))
	got, err := s.CalcAmountOut(in)
	require.NoError(t, err)
	require.NotEqual(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)

	planted := errors.New("planted")
	s.dCache.err = planted
	_, err = s.CalcAmountOut(in)
	require.ErrorIs(t, err, planted)
	_, err = s.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: *want.TokenAmountOut, TokenIn: s.Info.Tokens[0]})
	require.ErrorIs(t, err, planted)
}

// stable-meta-ng embeds this simulator and writes its reserves directly, without UpdateBalance: the
// cached D of the old reserves must then not be used.
func TestDCacheIgnoredAfterDirectReserveWrite(t *testing.T) {
	t.Parallel()
	s := newDCacheTestPool(t, dCacheTestPools[0], nil)
	s.Reserves[0].Add(&s.Reserves[0], s.Reserves[0].Clone())
	s.Info.Reserves[0] = s.Reserves[0].ToBig()
	requireCacheMiss(t, s)
	requireSameQuotes(t, s, rand.New(rand.NewPCG(3, 4)))
}

// While A ramps it changes with time, so a D cached at one A must not be used: quotes must follow A.
func TestDCacheNotUsedWhileARamps(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()
	s := newDCacheTestPool(t, dCacheTestPools[0], func(p *entity.Pool) {
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		extra.InitialA, extra.InitialATime = extra.FutureA.Clone(), now-3600
		extra.FutureA = new(uint256.Int).Mul(extra.FutureA, uint256.NewInt(10))
		extra.FutureATime = now + 3600
		b, err := json.Marshal(extra)
		require.NoError(t, err)
		p.Extra = string(b)
	})
	requireCacheMiss(t, s)
	requireSameQuotes(t, s, rand.New(rand.NewPCG(5, 6)))
}

func BenchmarkCalcAmountOutDCache(b *testing.B) {
	s := newDCacheTestPool(b, dCacheTestPools[0], nil)
	in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0], Amount: big.NewInt(1e18)},
		TokenOut: s.Info.Tokens[1]}
	for name, sim := range map[string]*PoolSimulator{"cached": s, "uncached": uncached(s)} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.CalcAmountOut(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
