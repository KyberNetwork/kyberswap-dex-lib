package gohm

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// indexAt25147546 is the observed on-chain index at block 25147546.
// sOHM.index() = 269238508004 (raw, 9-decimal fixed-point)
var indexAt25147546 = uint256.NewInt(269238508004)

const (
	testOlympusStakingAddress = "0xb63cac384247597756545b500253ff8e607a8020"
	testOHMAddress            = "0x64aa3364f17a4d01c6f1751fd97c2bd3d7e7f1d5"
	testSOHMAddress           = "0x04906695d6d12cf5459975d7c3c03356e4ccd460"
	testGOHMAddress           = "0x0ab87046fbb341d058f17cbc4c1133f25a20a52f"
)

// testOHMReserve and testSOHMReserve mirror the on-chain balances read at block 25148805.
// They are used as the default simulator reserves in tests that don't exercise cap logic.
var (
	testOHMReserve  = uint256.MustFromHex("0x3134B1EFD578") // 13862025744940277 (13.86M OHM, raw 9-dec)
	testSOHMReserve = uint256.MustFromHex("0x657C44AB7A3C") // 28443408722380364 (28.44M sOHM, raw 9-dec)
)

func newSimulator(t *testing.T, index *uint256.Int, warmupPeriod uint64) *PoolSimulator {
	t.Helper()
	return newSimulatorWithReserves(t, index, warmupPeriod, testOHMReserve, testSOHMReserve)
}

func newSimulatorWithReserves(
	t *testing.T,
	index *uint256.Int,
	warmupPeriod uint64,
	ohmReserve *uint256.Int,
	sohmReserve *uint256.Int,
) *PoolSimulator {
	t.Helper()
	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  testOlympusStakingAddress,
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{testOHMAddress, testSOHMAddress, testGOHMAddress},
			Reserves: []*big.Int{
				ohmReserve.ToBig(),
				sohmReserve.ToBig(),
				big.NewInt(0),
			},
		},
		index:        index,
		warmupPeriod: warmupPeriod,
		ohmReserve:   new(uint256.Int).Set(ohmReserve),
		sohmReserve:  new(uint256.Int).Set(sohmReserve),
	}
}

// ---- math unit tests ----

func TestBalanceFrom_OneGOHM(t *testing.T) {
	// 1 gOHM (1e18 raw) -> sOHM at index 269238508004
	// result = 1e18 * 269238508004 / 1e18 = 269238508004 raw = 269.238508004 sOHM
	oneGOHM := new(uint256.Int).Mul(uint256.NewInt(1), new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18)))
	result, err := balanceFrom(oneGOHM, indexAt25147546)
	require.NoError(t, err)
	assert.Equal(t, indexAt25147546, result)
}

func TestBalanceTo_OneOHM(t *testing.T) {
	// 1 OHM (1e9 raw) -> gOHM at index 269238508004
	// result = 1e9 * 1e18 / 269238508004 = 3714178953870682
	oneOHM := uint256.NewInt(1_000_000_000)
	result, err := balanceTo(oneOHM, indexAt25147546)
	require.NoError(t, err)
	expected := new(uint256.Int).Div(
		new(uint256.Int).Mul(oneOHM, number_1e18),
		indexAt25147546,
	)
	assert.Equal(t, expected, result)
}

func TestBalanceFromTo_RoundTrip(t *testing.T) {
	// 100 gOHM -> sOHM -> gOHM: should lose at most 1 unit due to integer division
	hundred_gohm := new(uint256.Int).Mul(
		uint256.NewInt(100),
		new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18)),
	)
	sohm, err := balanceFrom(hundred_gohm, indexAt25147546)
	require.NoError(t, err)
	gohm_back, err := balanceTo(sohm, indexAt25147546)
	require.NoError(t, err)

	diff := new(uint256.Int).Sub(hundred_gohm, gohm_back)
	assert.True(t, diff.IsZero() || diff.Eq(uint256.NewInt(1)),
		"round-trip drift must be 0 or 1, got %s", diff)
}

func TestIndexZero_Errors(t *testing.T) {
	_, err := balanceFrom(uint256.NewInt(1e18), uint256.NewInt(0))
	assert.ErrorIs(t, err, ErrIndexZero)
	_, err = balanceTo(uint256.NewInt(1e9), uint256.NewInt(0))
	assert.ErrorIs(t, err, ErrIndexZero)
}

// ---- simulator CalcAmountOut tests ----

func TestCalcAmountOut_OHMtoSOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	amtIn := big.NewInt(1_000_000_000) // 1 OHM
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: amtIn},
		TokenOut:      testSOHMAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "OHM->sOHM must be 1:1")
	assert.Equal(t, big.NewInt(0), res.Fee.Amount)
}

func TestCalcAmountOut_SOHMtoOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	amtIn := big.NewInt(500_000_000) // 0.5 sOHM
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSOHMAddress, Amount: amtIn},
		TokenOut:      testOHMAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, amtIn, res.TokenAmountOut.Amount, "sOHM->OHM must be 1:1")
}

func TestCalcAmountOut_OHMtoGOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	// 1 OHM -> gOHM
	amtIn := big.NewInt(1_000_000_000)
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: amtIn},
		TokenOut:      testGOHMAddress,
	})
	require.NoError(t, err)

	// expected = 1e9 * 1e18 / 269238508004 = 3714178953870682
	expected := new(big.Int).Div(
		new(big.Int).Mul(amtIn, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		big.NewInt(269238508004),
	)
	assert.Equal(t, expected, res.TokenAmountOut.Amount)
}

func TestCalcAmountOut_GOHMtoOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	// 1 gOHM -> OHM
	oneGOHM := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testGOHMAddress, Amount: oneGOHM},
		TokenOut:      testOHMAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(269238508004), res.TokenAmountOut.Amount)
}

func TestCalcAmountOut_GOHMtoSOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	oneGOHM := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testGOHMAddress, Amount: oneGOHM},
		TokenOut:      testSOHMAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(269238508004), res.TokenAmountOut.Amount)
}

func TestCalcAmountOut_SOHMtoGOHM(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	amtIn := big.NewInt(269238508004) // 1 gOHM worth of sOHM
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSOHMAddress, Amount: amtIn},
		TokenOut:      testGOHMAddress,
	})
	require.NoError(t, err)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	diff := new(big.Int).Sub(expected, res.TokenAmountOut.Amount)
	assert.True(t, diff.Int64() == 0 || diff.Int64() == 1,
		"should get ~1e18 gOHM back, diff=%s", diff)
}

func TestCalcAmountOut_WarmupBlocks_IndexSwaps(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 1) // warmupPeriod active
	amtIn := big.NewInt(1_000_000_000)

	// 1:1 swaps should still work
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: amtIn},
		TokenOut:      testSOHMAddress,
	})
	assert.NoError(t, err, "OHM->sOHM 1:1 must work with warmup active")

	// Index-based swaps blocked
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: amtIn},
		TokenOut:      testGOHMAddress,
	})
	assert.ErrorIs(t, err, ErrWarmupActive)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	amtIn := big.NewInt(1_000_000_000)

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", Amount: amtIn},
		TokenOut:      testOHMAddress,
	})
	assert.ErrorIs(t, err, ErrInvalidTokenIn)

	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: amtIn},
		TokenOut:      "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	})
	assert.ErrorIs(t, err, ErrInvalidTokenOut)
}

func TestCalcAmountOut_ZeroAmount(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: big.NewInt(0)},
		TokenOut:      testGOHMAddress,
	})
	assert.ErrorIs(t, err, ErrZeroAmount)
}

func TestCloneState_DeepCopy(t *testing.T) {
	s := newSimulator(t, indexAt25147546, 0)
	clone := s.CloneState().(*PoolSimulator)

	// Mutate the clone's index — original must not change
	clone.index.SetUint64(1)
	assert.Equal(t, uint64(269238508004), s.index.Uint64(), "original index must be unmodified after clone mutation")

	// Mutate the original's Info.Reserves slice — clone must not change
	origReserve0 := new(big.Int).Set(s.Info.Reserves[0])
	s.Info.Reserves[0].Add(s.Info.Reserves[0], big.NewInt(999))
	assert.Equal(t, origReserve0, clone.Info.Reserves[0], "clone reserve must be unmodified after original mutation")

	// Mutate the clone's ohmReserve — original must not change
	origOHMReserve := new(uint256.Int).Set(s.ohmReserve)
	clone.ohmReserve.SetUint64(1)
	assert.Equal(t, origOHMReserve, s.ohmReserve, "original ohmReserve must be unmodified after clone mutation")

	// Mutate the clone's sohmReserve — original must not change
	origSOHMReserve := new(uint256.Int).Set(s.sohmReserve)
	clone.sohmReserve.SetUint64(1)
	assert.Equal(t, origSOHMReserve, s.sohmReserve, "original sohmReserve must be unmodified after clone mutation")
}

// ---- balance-cap tests ----

// smallReserve is 1000 raw units — well below any real swap amount.
// It lets us test over-cap without needing astronomically large amountIn values.
var smallReserve = uint256.NewInt(1000)

// TestCalcAmountOut_CapExceeded_OHMtoSOHM verifies that OHM->sOHM returns
// ErrInsufficientLiquidity when amountIn > SOHMReserve.
func TestCalcAmountOut_CapExceeded_OHMtoSOHM(t *testing.T) {
	// SOHMReserve = 1000; send 1001 OHM (which would require 1001 sOHM in staking).
	s := newSimulatorWithReserves(t, indexAt25147546, 0, testOHMReserve, smallReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: big.NewInt(1001)},
		TokenOut:      testSOHMAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_CapOK_OHMtoSOHM verifies that OHM->sOHM succeeds when
// amountIn == SOHMReserve (boundary: exactly at cap).
func TestCalcAmountOut_CapOK_OHMtoSOHM(t *testing.T) {
	s := newSimulatorWithReserves(t, indexAt25147546, 0, testOHMReserve, smallReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: big.NewInt(1000)},
		TokenOut:      testSOHMAddress,
	})
	assert.NoError(t, err, "amountIn == SOHMReserve must succeed")
}

// TestCalcAmountOut_CapExceeded_SOHMtoOHM verifies that sOHM->OHM returns
// ErrInsufficientLiquidity when amountIn > OHMReserve.
func TestCalcAmountOut_CapExceeded_SOHMtoOHM(t *testing.T) {
	// OHMReserve = 1000; send 1001 sOHM (which would require 1001 OHM in staking).
	s := newSimulatorWithReserves(t, indexAt25147546, 0, smallReserve, testSOHMReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSOHMAddress, Amount: big.NewInt(1001)},
		TokenOut:      testOHMAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_CapExceeded_GOHMtoOHM verifies that gOHM->OHM returns
// ErrInsufficientLiquidity when balanceFrom(amountIn) > OHMReserve.
// With index=269238508004 and OHMReserve=1000:
//
//	balanceFrom(1e18) = 269238508004 >> 1000, so 1 gOHM triggers the cap.
func TestCalcAmountOut_CapExceeded_GOHMtoOHM(t *testing.T) {
	s := newSimulatorWithReserves(t, indexAt25147546, 0, smallReserve, testSOHMReserve)
	oneGOHM := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testGOHMAddress, Amount: oneGOHM},
		TokenOut:      testOHMAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_CapExceeded_GOHMtoSOHM verifies that gOHM->sOHM returns
// ErrInsufficientLiquidity when balanceFrom(amountIn) > SOHMReserve.
func TestCalcAmountOut_CapExceeded_GOHMtoSOHM(t *testing.T) {
	s := newSimulatorWithReserves(t, indexAt25147546, 0, testOHMReserve, smallReserve)
	oneGOHM := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testGOHMAddress, Amount: oneGOHM},
		TokenOut:      testSOHMAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_NoCap_OHMtoGOHM verifies that OHM->gOHM is never capped by
// staking-contract balances. Even with both reserves set to zero, the swap succeeds.
func TestCalcAmountOut_NoCap_OHMtoGOHM(t *testing.T) {
	zeroReserve := uint256.NewInt(0)
	s := newSimulatorWithReserves(t, indexAt25147546, 0, zeroReserve, zeroReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testOHMAddress, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      testGOHMAddress,
	})
	assert.NoError(t, err, "OHM->gOHM must not be blocked by reserve cap")
}

// TestCalcAmountOut_NoCap_SOHMtoGOHM verifies that sOHM->gOHM is never capped by
// staking-contract balances. Even with both reserves set to zero, the swap succeeds.
func TestCalcAmountOut_NoCap_SOHMtoGOHM(t *testing.T) {
	zeroReserve := uint256.NewInt(0)
	s := newSimulatorWithReserves(t, indexAt25147546, 0, zeroReserve, zeroReserve)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testSOHMAddress, Amount: big.NewInt(269238508004)},
		TokenOut:      testGOHMAddress,
	})
	assert.NoError(t, err, "sOHM->gOHM must not be blocked by reserve cap")
}

// TestCalcAmountOut_ConcurrentSharedSimulator runs CalcAmountOut from many
// goroutines on one simulator, as the router does with a shared pool set.
// Each result must match the value computed on a private simulator; run with
// -race to catch writes to shared receiver state.
func TestCalcAmountOut_ConcurrentSharedSimulator(t *testing.T) {
	pairs := [][2]string{
		{testOHMAddress, testSOHMAddress},
		{testSOHMAddress, testOHMAddress},
		{testOHMAddress, testGOHMAddress},
		{testGOHMAddress, testOHMAddress},
		{testSOHMAddress, testGOHMAddress},
		{testGOHMAddress, testSOHMAddress},
	}
	type job struct {
		in, out string
		amt     *big.Int
		want    *big.Int
	}
	var jobs []job
	for i := range 64 {
		p := pairs[i%len(pairs)]
		amt := new(big.Int).Mul(big.NewInt(int64(1000+i*7919)), big.NewInt(1e6))
		res, err := newSimulator(t, indexAt25147546, 0).CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: p[0], Amount: amt},
			TokenOut:      p[1],
		})
		require.NoError(t, err)
		jobs = append(jobs, job{p[0], p[1], amt, res.TokenAmountOut.Amount})
	}

	shared := newSimulator(t, indexAt25147546, 0)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				for _, j := range jobs {
					res, err := shared.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: j.in, Amount: j.amt},
						TokenOut:      j.out,
					})
					if !assert.NoError(t, err) {
						return
					}
					if !assert.Equal(t, j.want.String(), res.TokenAmountOut.Amount.String(), "%s->%s", j.in, j.out) {
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

const identityGolden = "testdata/calc_amount_out_identity.golden"

// identityAmounts covers zero, dust, typical sizes, cap-hitting sizes and a
// value above 2^256-1.
func identityAmounts() []*big.Int {
	out := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(7)}
	for e := 3; e <= 33; e += 3 {
		v := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil)
		out = append(out, v, new(big.Int).Add(v, big.NewInt(int64(e*131))))
	}
	return append(out, new(big.Int).Lsh(big.NewInt(1), 256))
}

// TestCalcAmountOut_Identity pins CalcAmountOut results on the package
// fixtures (live, capped-reserve, zero-index and warmup simulators) so refactors of
// the arithmetic are checked to keep behaviour identical.
// Regenerate with UPDATE_GOLDEN=1.
func TestCalcAmountOut_Identity(t *testing.T) {
	sims := []struct {
		name string
		sim  *PoolSimulator
	}{
		{"live", newSimulator(t, indexAt25147546, 0)},
		{"capped", newSimulatorWithReserves(t, indexAt25147546, 0,
			uint256.NewInt(5_000_000), uint256.NewInt(3_000_000))},
		{"index0", newSimulator(t, uint256.NewInt(0), 0)},
		{"warmup", newSimulator(t, indexAt25147546, 3)},
	}
	tokens := []string{testOHMAddress, testSOHMAddress, testGOHMAddress}

	var b strings.Builder
	for _, sc := range sims {
		for _, in := range tokens {
			for _, out := range tokens {
				for _, amt := range identityAmounts() {
					res, err := sc.sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt},
						TokenOut:      out,
					})
					fmt.Fprintf(&b, "%s %s>%s %s => ", sc.name, in[:6], out[:6], amt)
					if err != nil {
						fmt.Fprintf(&b, "err %v\n", err)
						continue
					}
					fmt.Fprintf(&b, "%s gas %d %+v\n", res.TokenAmountOut.Amount, res.Gas, res.SwapInfo)
				}
			}
		}
	}

	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(identityGolden), 0o755))
		require.NoError(t, os.WriteFile(identityGolden, []byte(b.String()), 0o644))
	}
	want, err := os.ReadFile(identityGolden)
	require.NoError(t, err)
	require.Equal(t, string(want), b.String())
}
