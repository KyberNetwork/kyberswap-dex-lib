package velodromev1

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	tokenA = "0x00000000000000000000000000000000000000a1"
	tokenB = "0x00000000000000000000000000000000000000b2"
)

type balanceTestPool struct {
	name       string
	stable     bool
	decimals   [2]uint8
	reserves   [2]string
	feeNum     uint64
	feePrecise uint64
}

var balanceTestPools = []balanceTestPool{
	{"volatile 6/6", false, [2]uint8{6, 6}, [2]string{"1000000000000", "3000000000000"}, 5, 10000},
	{"volatile 18/6", false, [2]uint8{18, 6}, [2]string{"1000000000000000000000000", "1000000000000"}, 5, 10000},
	{"stable 6/6", true, [2]uint8{6, 6}, [2]string{"5000000000000", "4000000000000"}, 5, 10000},
	{"stable 18/6", true, [2]uint8{18, 6}, [2]string{"2000000000000000000000000", "2000000000000"}, 1, 2000},
}

func bi(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// newBalanceTestSimulator builds the simulator through the entity (JSON Extra) like the router does. balances nil
// means unknown.
func newBalanceTestSimulator(t testing.TB, tp balanceTestPool, reserves []*big.Int, balances []*big.Int) *PoolSimulator {
	t.Helper()
	return newBalanceTestSimulatorOn(t, "velodrome", tp, reserves, balances)
}

// newBalanceTestSimulatorOn is newBalanceTestSimulator for an exchange.
func newBalanceTestSimulatorOn(t testing.TB, exchange string, tp balanceTestPool, reserves []*big.Int, balances []*big.Int) *PoolSimulator {
	t.Helper()
	extra := PoolExtra{Fee: tp.feeNum}
	if balances != nil {
		extra.Balance0, extra.Balance1 = balances[0], balances[1]
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	staticExtra, err := json.Marshal(PoolStaticExtra{FeePrecision: tp.feePrecise, Stable: tp.stable})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0xpair",
		Exchange: exchange,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: tokenA, Swappable: true, Decimals: tp.decimals[0]},
			{Address: tokenB, Swappable: true, Decimals: tp.decimals[1]},
		},
		Reserves:    entity.PoolReserves{reserves[0].String(), reserves[1].String()},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtra),
	})
	require.NoError(t, err)
	return sim
}

func tokenOf(i int) string { return []string{tokenA, tokenB}[i] }

func quote(sim *PoolSimulator, in int, amountIn *big.Int) (*poolpkg.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: tokenOf(in), Amount: amountIn},
		TokenOut:      tokenOf(1 - in),
	})
}

func updateBalance(sim *PoolSimulator, in int, amountIn *big.Int, res *poolpkg.CalcAmountOutResult) {
	sim.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  poolpkg.TokenAmount{Token: tokenOf(in), Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})
}

func simReserves(sim *PoolSimulator) [2]*big.Int {
	return [2]*big.Int{sim.reserves[0].ToBig(), sim.reserves[1].ToBig()}
}

// referenceSwap is DystPair.swap after the optimistic transfer, written with math/big from the contract source
// (swap :395-434, _update0/_update1 :167-198, _k :501-513, SWAP_FEE fee = amountIn * feeNum / feePrecise), to check
// the simulator against.
func referenceSwap(tp balanceTestPool, reserves, balances [2]*big.Int, in int, amountIn, amountOut *big.Int) ([2]*big.Int, error) {
	out := 1 - in
	if amountOut.Cmp(reserves[out]) >= 0 {
		return [2]*big.Int{}, ErrInsufficientLiquidity
	}
	if balances[out].Cmp(amountOut) < 0 {
		return [2]*big.Int{}, ErrInsufficientBalance
	}
	var bal [2]*big.Int
	bal[in] = new(big.Int).Add(balances[in], amountIn)
	bal[out] = new(big.Int).Sub(balances[out], amountOut)
	outs := [2]*big.Int{new(big.Int), new(big.Int)}
	outs[out] = amountOut
	var credited [2]*big.Int
	for i := range bal {
		credited[i] = new(big.Int)
		if floor := new(big.Int).Sub(reserves[i], outs[i]); bal[i].Cmp(floor) > 0 {
			credited[i].Sub(bal[i], floor)
		}
	}
	if credited[0].Sign() == 0 && credited[1].Sign() == 0 {
		return [2]*big.Int{}, ErrInsufficientInputAmount
	}
	for i := range bal {
		fee := new(big.Int).Div(new(big.Int).Mul(credited[i], big.NewInt(int64(tp.feeNum))), big.NewInt(int64(tp.feePrecise)))
		bal[i] = new(big.Int).Sub(bal[i], fee)
	}
	k := func(x, y *big.Int) *big.Int {
		if !tp.stable {
			return new(big.Int).Mul(x, y)
		}
		e18 := big.NewInt(1e18)
		pow10 := func(d uint8) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil) }
		_x := new(big.Int).Div(new(big.Int).Mul(x, e18), pow10(tp.decimals[0]))
		_y := new(big.Int).Div(new(big.Int).Mul(y, e18), pow10(tp.decimals[1]))
		a := new(big.Int).Div(new(big.Int).Mul(_x, _y), e18)
		b := new(big.Int).Add(new(big.Int).Div(new(big.Int).Mul(_x, _x), e18), new(big.Int).Div(new(big.Int).Mul(_y, _y), e18))
		return new(big.Int).Div(new(big.Int).Mul(a, b), e18)
	}
	if k(bal[0], bal[1]).Cmp(k(reserves[0], reserves[1])) < 0 {
		return [2]*big.Int{}, ErrK
	}
	return bal, nil
}

// referenceExecute is ExecutorV3Helper1.executeVelodrome with math/big: the router has sent `sent` to the pair, the
// executor takes amountIn = balanceOf(pair) - reserveIn (unchecked, so a result below zero wraps and the swap reverts),
// amountOut = pair.getAmountOut(amountIn) and then calls pair.swap. getAmountOut is taken from quoteOf, the
// quote of the pair from its reserves (the legacy simulator without balances, which the parity tests pin to the
// chain). It returns the balances after the swap, the credited amount in and the amount out.
func referenceExecute(tp balanceTestPool, reserves, balances [2]*big.Int, in int, sent *big.Int,
	quoteOf func(amountIn *big.Int) (*big.Int, error)) (post [2]*big.Int, credited, amountOut *big.Int, err error) {
	credited = new(big.Int).Sub(new(big.Int).Add(balances[in], sent), reserves[in])
	if credited.Sign() <= 0 {
		return post, credited, nil, ErrInsufficientInputAmount
	}
	if amountOut, err = quoteOf(credited); err != nil {
		return post, credited, nil, err
	}
	post, err = referenceSwap(tp, reserves, balances, in, sent, amountOut)
	return post, credited, amountOut, err
}

func TestPoolSimulator_Balances_HealthyPoolIsQuotedAsBefore(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(1))
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			legacy := newBalanceTestSimulator(t, tp, reserves, nil)
			mirrored := newBalanceTestSimulator(t, tp, reserves, reserves)
			quoted := 0
			for range 400 {
				in := rng.Intn(2)
				// from 1 wei to a third of the reserve
				amountIn := new(big.Int).Rand(rng, new(big.Int).Div(reserves[in], big.NewInt(3)))
				amountIn.Add(amountIn, big.NewInt(1))
				if rng.Intn(2) == 0 {
					amountIn.Rsh(amountIn, uint(rng.Intn(40)))
					amountIn.Add(amountIn, big.NewInt(1))
				}
				want, wantErr := quote(legacy, in, amountIn)
				got, gotErr := quote(mirrored, in, amountIn)
				if wantErr != nil {
					// the mirror only adds rejections, never changes the quote
					continue
				}
				require.NoError(t, gotErr, "in=%d amountIn=%s", in, amountIn)
				assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
				assert.Equal(t, want.Fee.Amount, got.Fee.Amount)
				quoted++

				// and the exact-out side: the amount in it finds is accepted by the pair
				amountOut := want.TokenAmountOut.Amount
				if wantIn, err := legacy.CalcAmountIn(poolpkg.CalcAmountInParams{
					TokenAmountOut: poolpkg.TokenAmount{Token: tokenOf(1 - in), Amount: amountOut}, TokenIn: tokenOf(in)}); err == nil {
					gotIn, gotErr := mirrored.CalcAmountIn(poolpkg.CalcAmountInParams{
						TokenAmountOut: poolpkg.TokenAmount{Token: tokenOf(1 - in), Amount: amountOut}, TokenIn: tokenOf(in)})
					if gotErr == nil {
						assert.Equal(t, wantIn.TokenAmountIn.Amount, gotIn.TokenAmountIn.Amount)
					} else {
						t.Logf("CalcAmountIn rejected by the K check: in=%d amountOut=%s err=%v", in, amountOut, gotErr)
						assert.Fail(t, "CalcAmountIn of a healthy pool must pass the pair checks")
					}
				}
			}
			assert.Greater(t, quoted, 100)
		})
	}
}

// TestPoolSimulator_Balances_AgainstContractModel runs the simulator against referenceExecute on a grid of pool
// balances around the reserves (donations and deficits on either token) and swap sizes.
func TestPoolSimulator_Balances_AgainstContractModel(t *testing.T) {
	t.Parallel()
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := [2]*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			legacy := newBalanceTestSimulator(t, tp, reserves[:], nil)
			quoteOf := func(in int) func(*big.Int) (*big.Int, error) {
				return func(amountIn *big.Int) (*big.Int, error) {
					res, err := quote(legacy, in, amountIn)
					if err != nil {
						return nil, err
					}
					return res.TokenAmountOut.Amount, nil
				}
			}
			passed, failedByKind := 0, map[error]int{}
			shifts := func(r *big.Int) []*big.Int {
				d := func(div int64) *big.Int { return new(big.Int).Div(r, big.NewInt(div)) }
				return []*big.Int{new(big.Int).Neg(r), new(big.Int).Neg(d(100)), big.NewInt(-1000), big.NewInt(-1),
					big.NewInt(0), big.NewInt(1), big.NewInt(1000), d(100)}
			}
			for in := range 2 {
				for _, div := range []int64{1000, 10, 3} {
					sent := new(big.Int).Div(reserves[in], big.NewInt(div))
					for _, d0 := range shifts(reserves[0]) {
						for _, d1 := range shifts(reserves[1]) {
							balances := [2]*big.Int{new(big.Int).Add(reserves[0], d0), new(big.Int).Add(reserves[1], d1)}
							if balances[0].Sign() < 0 || balances[1].Sign() < 0 {
								continue
							}
							name := fmt.Sprintf("in=%d sent=%s d0=%s d1=%s", in, sent, d0, d1)
							want, credited, wantOut, wantErr := referenceExecute(tp, reserves, balances, in, sent, quoteOf(in))
							sim := newBalanceTestSimulator(t, tp, reserves[:], balances[:])
							got, gotErr := quote(sim, in, sent)
							if wantErr != nil {
								assert.ErrorIs(t, gotErr, wantErr, name)
								failedByKind[wantErr]++
								continue
							}
							require.NoError(t, gotErr, name)
							// the quote is from the credited amount in, the fee is taken on it
							assert.Equal(t, wantOut, got.TokenAmountOut.Amount, name)
							wantFee := new(big.Int).Div(new(big.Int).Mul(credited, big.NewInt(int64(tp.feeNum))), big.NewInt(int64(tp.feePrecise)))
							assert.Equal(t, wantFee, got.Fee.Amount, name)
							updateBalance(sim, in, sent, got)
							// the pair syncs its reserves to the balances after the swap
							assert.Equal(t, [2]*big.Int{want[0], want[1]}, simReserves(sim), name)
							assert.Equal(t, []*big.Int{want[0], want[1]}, []*big.Int{sim.balances[0].ToBig(), sim.balances[1].ToBig()}, name)
							passed++
						}
					}
				}
			}
			t.Logf("passed=%d rejected=%v", passed, failedByKind)
			assert.Positive(t, passed)
			for _, kind := range []error{ErrInsufficientBalance, ErrK, ErrInsufficientInputAmount} {
				assert.Positive(t, failedByKind[kind], kind)
			}
		})
	}
}

func TestPoolSimulator_Balances(t *testing.T) {
	t.Parallel()
	vol := balanceTestPools[0]
	reserves := []*big.Int{bi(vol.reserves[0]), bi(vol.reserves[1])}
	amountIn := big.NewInt(1_000_000_000) // 1000 units of 6 decimals
	legacyQuote, err := quote(newBalanceTestSimulator(t, vol, reserves, nil), 0, amountIn)
	require.NoError(t, err)
	amountOut := legacyQuote.TokenAmountOut.Amount

	// the executor quotes from balanceIn + sent - reserveIn: a donation on the input side is credited to the swapper
	t.Run("donation on the input side is quoted as part of the amount in", func(t *testing.T) {
		donation := big.NewInt(5_000_000)
		donated := []*big.Int{new(big.Int).Add(reserves[0], donation), reserves[1]}
		res, err := quote(newBalanceTestSimulator(t, vol, reserves, donated), 0, amountIn)
		require.NoError(t, err)
		credited := new(big.Int).Add(amountIn, donation)
		want, err := quote(newBalanceTestSimulator(t, vol, reserves, nil), 0, credited)
		require.NoError(t, err)
		assert.Equal(t, want.TokenAmountOut.Amount, res.TokenAmountOut.Amount)
		assert.Equal(t, want.Fee.Amount, res.Fee.Amount) // the fee is on the credited amount
		assert.Equal(t, 1, res.TokenAmountOut.Amount.Cmp(amountOut), "more than the quote from the amount sent")
	})

	t.Run("donation on the output side does not change the quote", func(t *testing.T) {
		donated := []*big.Int{reserves[0], new(big.Int).Add(reserves[1], big.NewInt(5_000_000))}
		res, err := quote(newBalanceTestSimulator(t, vol, reserves, donated), 0, amountIn)
		require.NoError(t, err)
		assert.Equal(t, amountOut, res.TokenAmountOut.Amount)
	})

	t.Run("tokenOut balance below the amount out", func(t *testing.T) {
		for _, tc := range []struct {
			bal1 *big.Int
			want error
		}{
			{new(big.Int).Sub(amountOut, big.NewInt(1)), ErrInsufficientBalance},
			{big.NewInt(0), ErrInsufficientBalance},
			// holds the amount out but not the reserve: the K check rejects it
			{amountOut, ErrK},
		} {
			_, err := quote(newBalanceTestSimulator(t, vol, reserves, []*big.Int{reserves[0], tc.bal1}), 0, amountIn)
			assert.ErrorIs(t, err, tc.want, tc.bal1.String())
		}
	})

	// the executor credits balanceIn + sent - reserveIn, so an input-side deficit eats into the amount in and the
	// swap succeeds with the smaller output, as long as something is left to credit
	t.Run("input-side deficit is quoted from the amount in less the deficit", func(t *testing.T) {
		tp := balanceTestPools[1] // volatile 18/6
		res18 := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
		in18 := bi("1000000000000000000000") // 1000 token0
		legacy := newBalanceTestSimulator(t, tp, res18, nil)
		quoteWithDeficit := func(d *big.Int) (*poolpkg.CalcAmountOutResult, error) {
			bal := []*big.Int{new(big.Int).Sub(res18[0], d), res18[1]}
			return quote(newBalanceTestSimulator(t, tp, res18, bal), 0, in18)
		}
		for _, d := range []*big.Int{big.NewInt(1), bi("1000000000000"), bi("500000000000000000000"), bi("999000000000000000000")} {
			res, err := quoteWithDeficit(d)
			require.NoError(t, err, d.String())
			want, err := quote(legacy, 0, new(big.Int).Sub(in18, d))
			require.NoError(t, err, d.String())
			assert.Equal(t, want.TokenAmountOut.Amount, res.TokenAmountOut.Amount, d.String())
			assert.Equal(t, want.Fee.Amount, res.Fee.Amount, d.String())
			assert.NotNil(t, res.SwapInfo, d.String())
		}
		// the deficit takes all of the amount in, or more: the executor's subtraction wraps and the swap reverts
		for _, d := range []*big.Int{in18, new(big.Int).Add(in18, big.NewInt(1)), res18[0]} {
			_, err := quoteWithDeficit(d)
			assert.ErrorIs(t, err, ErrInsufficientInputAmount, d.String())
		}
	})

	t.Run("output-side deficit is rejected whatever the input side holds", func(t *testing.T) {
		// token1 is 1 wei short: the curve rejects it, a donation of token0 is spent by the quote and does not pay for it
		short := new(big.Int).Sub(reserves[1], big.NewInt(1))
		for _, bal0 := range []*big.Int{reserves[0], new(big.Int).Add(reserves[0], big.NewInt(1_000_000))} {
			_, err := quote(newBalanceTestSimulator(t, vol, reserves, []*big.Int{bal0, short}), 0, amountIn)
			assert.ErrorIs(t, err, ErrK, bal0.String())
		}
	})

	// dystopia 0x421a018cc5839c4c0300afb21c725776dc389b1a on polygon: reserves report 999.5M USD+ but the pair
	// holds none, and USDC.e is full
	t.Run("dystopia shape: USD+ balance 0 with reserves of 999.5M", func(t *testing.T) {
		tp := balanceTestPool{"dystopia", true, [2]uint8{6, 6}, [2]string{"999500000000000", "730000"}, 5, 10000}
		res := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
		bals := []*big.Int{big.NewInt(0), big.NewInt(730_000)}
		sim := newBalanceTestSimulator(t, tp, res, bals)
		// the reserves are the on-chain ones, not zeroed
		assert.Equal(t, res[0], sim.reserves[0].ToBig())

		// USDC.e in, USD+ out: the pair cannot pay the USD+
		_, err := quote(sim, 1, big.NewInt(200_000))
		assert.ErrorIs(t, err, ErrInsufficientBalance)
		// USD+ in, USDC.e out: nothing is credited, the balance of USD+ is far below its reserve
		usdPlusIn := big.NewInt(1_000_000_000) // 1000 USD+, enough for a non-zero amount out of this skewed curve
		_, err = quote(sim, 0, usdPlusIn)
		assert.ErrorIs(t, err, ErrInsufficientInputAmount)
		// the same pair with its balances is quoted
		full := newBalanceTestSimulator(t, tp, res, res)
		_, err = quote(full, 1, big.NewInt(200_000))
		assert.NoError(t, err)
		_, err = quote(full, 0, usdPlusIn)
		assert.NoError(t, err)
	})
}

func TestPoolSimulator_Balances_UpdateBalanceChain(t *testing.T) {
	t.Parallel()
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			legacy := newBalanceTestSimulator(t, tp, reserves, nil)
			mirrored := newBalanceTestSimulator(t, tp, reserves, reserves)
			cloned := mirrored.CloneState().(*PoolSimulator)

			amounts := []struct {
				in     int
				amount *big.Int
			}{{0, new(big.Int).Div(reserves[0], big.NewInt(50))}, {0, new(big.Int).Div(reserves[0], big.NewInt(70))},
				{1, new(big.Int).Div(reserves[1], big.NewInt(60))}, {1, new(big.Int).Div(reserves[1], big.NewInt(20))}}
			for _, hop := range amounts {
				want, err := quote(legacy, hop.in, hop.amount)
				require.NoError(t, err)
				got, err := quote(mirrored, hop.in, hop.amount)
				require.NoError(t, err)
				assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
				updateBalance(legacy, hop.in, hop.amount, want)
				updateBalance(mirrored, hop.in, hop.amount, got)
				// with balances == reserves the mirror ends where the legacy update ends
				assert.Equal(t, simReserves(legacy), simReserves(mirrored))
				assert.Equal(t, simReserves(mirrored), [2]*big.Int{mirrored.balances[0].ToBig(), mirrored.balances[1].ToBig()})
			}
			// the clone did not move
			assert.Equal(t, [2]*big.Int{reserves[0], reserves[1]}, simReserves(cloned))
			assert.Equal(t, reserves[0], cloned.balances[0].ToBig())
		})
	}

	t.Run("a donation becomes part of the reserves after the first swap", func(t *testing.T) {
		tp := balanceTestPools[0]
		reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
		donation := big.NewInt(7_000_000)
		donated := []*big.Int{reserves[0], new(big.Int).Add(reserves[1], donation)}
		sim := newBalanceTestSimulator(t, tp, reserves, donated)
		amountIn := big.NewInt(1_000_000_000)
		res, err := quote(sim, 0, amountIn)
		require.NoError(t, err)
		updateBalance(sim, 0, amountIn, res)
		// reserves = balances = old balances moved by the swap, minus the fee of the input and of the donation
		fee0 := new(big.Int).Div(new(big.Int).Mul(amountIn, big.NewInt(5)), big.NewInt(10000))
		fee1 := new(big.Int).Div(new(big.Int).Mul(donation, big.NewInt(5)), big.NewInt(10000))
		want0 := new(big.Int).Sub(new(big.Int).Add(reserves[0], amountIn), fee0)
		want1 := new(big.Int).Sub(new(big.Int).Sub(donated[1], res.TokenAmountOut.Amount), fee1)
		assert.Equal(t, [2]*big.Int{want0, want1}, simReserves(sim))
	})

	t.Run("a swap the pair rejects stops the balances being trusted", func(t *testing.T) {
		tp := balanceTestPools[0]
		reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
		sim := newBalanceTestSimulator(t, tp, reserves, reserves)
		sim.UpdateBalance(poolpkg.UpdateBalanceParams{
			TokenAmountIn:  poolpkg.TokenAmount{Token: tokenA, Amount: big.NewInt(1_000_000)},
			TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: reserves[1]}, // all of the reserve
			Fee:            poolpkg.TokenAmount{Token: tokenA, Amount: big.NewInt(500)},
		})
		assert.Nil(t, sim.balances)
	})
}

func TestPoolSimulator_Balances_Unknown(t *testing.T) {
	t.Parallel()
	tp := balanceTestPools[2]
	reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
	amountIn := big.NewInt(1_000_000_000)
	want, err := quote(newBalanceTestSimulator(t, tp, reserves, nil), 0, amountIn)
	require.NoError(t, err)

	// an extra saved before the balances existed, or with only one of them
	for name, extra := range map[string]string{
		"old extra":          `{"fee":5}`,
		"only balance0":      `{"fee":5,"balance0":0}`,
		"only balance1":      `{"fee":5,"balance1":0}`,
		"null balances":      `{"fee":5,"balance0":null,"balance1":null}`,
		"balances == zeroes": ``,
	} {
		if extra == `` {
			continue
		}
		sim, err := NewPoolSimulator(entity.Pool{
			Tokens:   []*entity.PoolToken{{Address: tokenA, Decimals: 6}, {Address: tokenB, Decimals: 6}},
			Reserves: entity.PoolReserves{reserves[0].String(), reserves[1].String()}, Extra: extra,
			StaticExtra: `{"feePrecision":10000,"stable":true}`,
		})
		require.NoError(t, err, name)
		assert.Nil(t, sim.balances, name)
		got, err := quote(sim, 0, amountIn)
		require.NoError(t, err, name)
		assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount, name)
		updateBalance(sim, 0, amountIn, got)
		assert.Nil(t, sim.balances, name)
	}

	// Extra with balances round-trips through JSON
	b, err := json.Marshal(PoolExtra{Fee: 5, Balance0: big.NewInt(0), Balance1: big.NewInt(9)})
	require.NoError(t, err)
	var back PoolExtra
	require.NoError(t, json.Unmarshal(b, &back))
	assert.Equal(t, 0, back.Balance0.Sign())
	assert.Equal(t, big.NewInt(9), back.Balance1)
	assert.NotContains(t, string(mustJSON(t, PoolExtra{Fee: 5})), "balance")
	_ = errors.New
}

func mustJSON(t testing.TB, v any) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// TestPoolSimulator_Balances_CalcAmountIn: the router sends what the pair needs credited less its input-side surplus
// (plus its deficit), and the amount it finds is quoted back to the same output by CalcAmountOut.
func TestPoolSimulator_Balances_CalcAmountIn(t *testing.T) {
	t.Parallel()
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := [2]*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			legacy := newBalanceTestSimulator(t, tp, reserves[:], nil)
			for in := range 2 {
				out := 1 - in
				amountOut := new(big.Int).Div(reserves[out], big.NewInt(50))
				params := poolpkg.CalcAmountInParams{
					TokenAmountOut: poolpkg.TokenAmount{Token: tokenOf(out), Amount: amountOut}, TokenIn: tokenOf(in)}
				needed, err := legacy.CalcAmountIn(params) // what the pair has to credit
				require.NoError(t, err)
				credited := needed.TokenAmountIn.Amount
				want, err := quote(legacy, in, credited)
				require.NoError(t, err)

				for name, delta := range map[string]*big.Int{ // balanceIn - reserveIn
					"healthy":                   big.NewInt(0),
					"input-side donation":       new(big.Int).Div(credited, big.NewInt(3)),
					"input-side small deficit":  big.NewInt(-3),
					"input-side larger deficit": new(big.Int).Neg(new(big.Int).Div(credited, big.NewInt(2))),
				} {
					balances := reserves
					balances[in] = new(big.Int).Add(reserves[in], delta)
					sim := newBalanceTestSimulator(t, tp, reserves[:], balances[:])
					res, err := sim.CalcAmountIn(params)
					require.NoError(t, err, name)
					sent := new(big.Int).Sub(credited, delta)
					assert.Equal(t, sent, res.TokenAmountIn.Amount, name)
					_, ok := res.SwapInfo.(SwapInfo)
					assert.True(t, ok, name)

					// sending it, the executor credits what was needed and pays the same output
					got, err := quote(sim, in, res.TokenAmountIn.Amount)
					require.NoError(t, err, name)
					assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount, name)
				}

				// the pair already credits what is needed: nothing is left to send
				balances := reserves
				balances[in] = new(big.Int).Add(reserves[in], credited)
				_, err = newBalanceTestSimulator(t, tp, reserves[:], balances[:]).CalcAmountIn(params)
				assert.ErrorIs(t, err, ErrInsufficientInputAmount)
			}
		})
	}
}

// TestPoolSimulator_Balances_ViaRouter: hyperpie-v2 and aeon-v2 swap through their own router, which quotes from
// the amount sent: a donation is not credited and a deficit is not netted off the amount in.
func TestPoolSimulator_Balances_ViaRouter(t *testing.T) {
	t.Parallel()
	tp := balanceTestPools[1] // volatile 18/6
	reserves := []*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
	sent := bi("1000000000000000000000") // 1000 token0
	legacy := newBalanceTestSimulator(t, tp, reserves, nil)
	want, err := quote(legacy, 0, sent)
	require.NoError(t, err)
	wantIn, err := legacy.CalcAmountIn(poolpkg.CalcAmountInParams{
		TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: want.TokenAmountOut.Amount}, TokenIn: tokenA})
	require.NoError(t, err)

	for _, exchange := range []string{"hyperpie-v2", "aeon-v2"} {
		t.Run(exchange, func(t *testing.T) {
			donated := []*big.Int{new(big.Int).Add(reserves[0], bi("5000000000000000000")), reserves[1]}
			sim := newBalanceTestSimulatorOn(t, exchange, tp, reserves, donated)
			res, err := quote(sim, 0, sent)
			require.NoError(t, err)
			assert.Equal(t, want.TokenAmountOut.Amount, res.TokenAmountOut.Amount)
			assert.Equal(t, want.Fee.Amount, res.Fee.Amount)
			resIn, err := sim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: tokenB, Amount: want.TokenAmountOut.Amount}, TokenIn: tokenA})
			require.NoError(t, err)
			assert.Equal(t, wantIn.TokenAmountIn.Amount, resIn.TokenAmountIn.Amount)

			// half of the amount sent is missing from the input side: the quote from the amount sent fails K
			short := []*big.Int{new(big.Int).Sub(reserves[0], new(big.Int).Div(sent, big.NewInt(2))), reserves[1]}
			_, err = quote(newBalanceTestSimulatorOn(t, exchange, tp, reserves, short), 0, sent)
			assert.ErrorIs(t, err, ErrK)
			// while the exchanges that the executor swaps with directly quote the smaller credited amount
			_, err = quote(newBalanceTestSimulator(t, tp, reserves, short), 0, sent)
			assert.NoError(t, err)
		})
	}
}

// TestPoolSimulator_Balances_DonationChain follows the pair through two swaps in a row, the second one after the
// first donation was folded into the reserves, against the executor model.
func TestPoolSimulator_Balances_DonationChain(t *testing.T) {
	t.Parallel()
	for _, tp := range balanceTestPools {
		t.Run(tp.name, func(t *testing.T) {
			reserves := [2]*big.Int{bi(tp.reserves[0]), bi(tp.reserves[1])}
			balances := [2]*big.Int{
				new(big.Int).Add(reserves[0], new(big.Int).Div(reserves[0], big.NewInt(1000))),
				new(big.Int).Sub(reserves[1], big.NewInt(1)).Add(reserves[1], big.NewInt(5)), // reserve + 4
			}
			sim := newBalanceTestSimulator(t, tp, reserves[:], balances[:])
			for _, hop := range []struct {
				in   int
				sent *big.Int
			}{{0, new(big.Int).Div(reserves[0], big.NewInt(70))}, {1, new(big.Int).Div(reserves[1], big.NewInt(60))},
				{0, new(big.Int).Div(reserves[0], big.NewInt(90))}} {
				// the pair as the model has it: the reserves are the balances synced after each swap
				legacy := newBalanceTestSimulator(t, tp, reserves[:], nil)
				quoteOf := func(amountIn *big.Int) (*big.Int, error) {
					res, err := quote(legacy, hop.in, amountIn)
					if err != nil {
						return nil, err
					}
					return res.TokenAmountOut.Amount, nil
				}
				want, _, wantOut, err := referenceExecute(tp, reserves, balances, hop.in, hop.sent, quoteOf)
				require.NoError(t, err)
				res, err := quote(sim, hop.in, hop.sent)
				require.NoError(t, err)
				assert.Equal(t, wantOut, res.TokenAmountOut.Amount)
				updateBalance(sim, hop.in, hop.sent, res)
				reserves, balances = want, want
				assert.Equal(t, reserves, simReserves(sim))
				assert.Equal(t, balances, [2]*big.Int{sim.balances[0].ToBig(), sim.balances[1].ToBig()})
			}
		})
	}
}
