package doppleramm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func bi(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}

// Every expected amount below was measured against the deployed DopplerQuoter
// on a Base Sepolia fork, with a CPMM market bootstrapped at 100e18 / 200000e18
// and swapFeeBps = 30 while protocolFeeForMarket returned base = 10. A change
// to the fee order or the CPMM rounding breaks these to the wei.
func TestCalcAmountOutMatchesDeployedQuoter(t *testing.T) {
	sim := newTestSimulator(t, "100000000000000000000", "200000000000000000000000", 10, 30)

	xToY := []struct{ in, out string }{
		{"1000000000000000", "1991986159758089224"},
		{"10000000000000000", "19918076153639674642"},
		{"100000000000000000", "199002393019545353769"},
		{"500000000000000000", "991067469137682305055"},
		{"1000000000000000000", "1972361223047609121719"},
		{"10000000000000000000", "18115728051365573472469"},
		{"50000000000000000000", "66488785224847905693018"},
	}
	for _, tc := range xToY {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: bi(tc.in)},
			TokenOut:      sim.Info.Tokens[1],
		})
		require.NoError(t, err)
		require.Equal(t, tc.out, res.TokenAmountOut.Amount.String(), "X->Y in %s", tc.in)
		// The encoder reads tokenX off SwapInfo to derive inputIsX.
		require.Equal(t, sim.extra, res.SwapInfo)
		require.NotEmpty(t, res.SwapInfo.(Extra).TokenX)
	}

	yToX := []struct{ in, out string }{
		{"2000000000000000000", "995993079879044"},
		{"20000000000000000000", "9959038076819837"},
		{"200000000000000000000", "99501196509772676"},
		{"1000000000000000000000", "495533734568841152"},
		{"2000000000000000000000", "986180611523804560"},
		{"20000000000000000000000", "9057864025682786736"},
		{"100000000000000000000000", "33244392612423952846"},
	}
	for _, tc := range yToX {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: bi(tc.in)},
			TokenOut:      sim.Info.Tokens[0],
		})
		require.NoError(t, err)
		require.Equal(t, tc.out, res.TokenAmountOut.Amount.String(), "Y->X in %s", tc.in)
	}
}

func TestDeriveLpFeeBpsFromProbe(t *testing.T) {
	// 1e18 X in quoted 1972361223047609121719 Y out on the fork.
	lp, err := deriveLpFeeBps(
		bi("1000000000000000000"),
		bi("1972361223047609121719"),
		bi("100000000000000000000"),
		bi("200000000000000000000000"),
		10,
	)
	require.NoError(t, err)
	require.Equal(t, uint16(30), lp)

	// A probe that cannot be explained by any fee must be rejected rather than
	// silently persisted as a wrong rate.
	_, err = deriveLpFeeBps(
		bi("1000000000000000000"),
		bi("1000000000000000000"),
		bi("100000000000000000000"),
		bi("200000000000000000000000"),
		10,
	)
	require.ErrorIs(t, err, ErrFeeDerivation)

	// Output at or above the reserve is not a valid quote.
	_, err = deriveLpFeeBps(bi("1000000000000000000"), bi("200000000000000000000000"),
		bi("100000000000000000000"), bi("200000000000000000000000"), 10)
	require.ErrorIs(t, err, ErrFeeDerivation)
}

// Real market 0xcab05a53... on Base (WETH/USDC, LP fee 5 bps, base fee 10 bps).
// Every quoted output below came from DopplerQuoter.quoteSwapExactInput at block
// 56.3M, including the amount whose output floors to zero.
func TestDeriveLpFeeBpsRealMarket(t *testing.T) {
	reserveX := bi("2275146628267141")
	reserveY := bi("4777975")

	// 1% of reserveX pins the rate down.
	lp, err := deriveLpFeeBps(bi("22751466282671"), bi("47236"), reserveX, reserveY, 10)
	require.NoError(t, err)
	require.Equal(t, uint16(5), lp)

	// 0.1% of reserveX is too small: 4 bps and 5 bps both reproduce the quote, so
	// the answer must be reported as unresolvable instead of guessed.
	_, err = deriveLpFeeBps(bi("2275146628267"), bi("4766"), reserveX, reserveY, 10)
	require.ErrorIs(t, err, ErrFeeDerivation)

	// The derived rate reproduces the whole quote curve, not just the probe.
	for _, tc := range []struct{ in, out string }{
		{"100000000000", "209"},
		{"1000000000000", "2096"},
		{"2275146628267", "4766"},
		{"227514662826714", "433769"},
		{"476889019", "1"},
		{"476889018", "0"},
	} {
		in, _ := uint256.FromBig(bi(tc.in))
		rx, _ := uint256.FromBig(reserveX)
		ry, _ := uint256.FromBig(reserveY)
		out, err := getAmountOut(in, rx, ry, 10, lp)
		require.NoError(t, err)
		require.Equal(t, tc.out, out.Dec(), "X->Y in %s", tc.in)
	}
}

func TestCalcAmountOutRejectsUnknownTokenAndZeroAmount(t *testing.T) {
	sim := newTestSimulator(t, "100000000000000000000", "200000000000000000000000", 10, 30)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x0000000000000000000000000000000000000099", Amount: big.NewInt(1e18)},
		TokenOut:      sim.Info.Tokens[1],
	})
	require.ErrorIs(t, err, ErrInvalidToken)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(0)},
		TokenOut:      sim.Info.Tokens[1],
	})
	require.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestUpdateBalanceMovesReservesAndCloneIsolates(t *testing.T) {
	sim := newTestSimulator(t, "100000000000000000000", "200000000000000000000000", 10, 30)
	clone := sim.CloneState().(*PoolSimulator)

	amountIn, amountOut := big.NewInt(1e18), bi("1972361223047609121719")
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: amountIn},
		TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: amountOut},
	})

	require.Equal(t, "101000000000000000000", sim.reserves[0].Dec())
	require.Equal(t, "198027638776952390878281", sim.reserves[1].Dec())
	require.Equal(t, "100000000000000000000", clone.reserves[0].Dec(),
		"clone must keep the pre-swap reserves")
}

func TestGetMetaInfoApprovesAMM(t *testing.T) {
	sim := newTestSimulator(t, "100000000000000000000", "200000000000000000000000", 10, 30)
	meta, ok := sim.GetMetaInfo("", "").(pool.MetaInfo)
	require.True(t, ok)
	require.Equal(t, "0x00000000000000000000000000000000000000aa", meta.ApprovalAddress)
}

func newTestSimulator(t *testing.T, reserveX, reserveY string, baseFeeBps, lpFeeBps uint16) *PoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{
		TokenX:     "0x0000000000000000000000000000000000000011",
		BaseFeeBps: baseFeeBps,
		LpFeeBps:   lpFeeBps,
	})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{
		MarketID: "0x01",
		TokenX:   "0x0000000000000000000000000000000000000011",
		TokenY:   "0x0000000000000000000000000000000000000022",
		AMM:      "0x00000000000000000000000000000000000000aa",
		Quoter:   "0x00000000000000000000000000000000000000bb",
	})
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  "0x01",
		Exchange: "doppler-amm",
		Type:     DexType,
		Reserves: entity.PoolReserves{reserveX, reserveY},
		Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000011", Swappable: true},
			{Address: "0x0000000000000000000000000000000000000022", Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	return sim
}

// Base txs 0x9bf40413... (buy) and 0x2c5bb90f... (sell) on market 0xbf674fb6...,
// base fee 10 bps, LP fee 100 bps. The sell was quoted ~9x low because the pool
// still held its pre-buy reserves; the chain output is wei-exact once the buy has
// been applied (reserves refreshed from chain). Fixtures are archive reads at blocks 56572972 and 56574352.
func TestSellAfterBuyMatchesChain(t *testing.T) {
	sim := newTestSimulator(t, "9888682694008172885300", "24227272600320", 10, 100)
	tokenX, tokenY := sim.Info.Tokens[0], sim.Info.Tokens[1]

	buyIn := bi("100000000000000")
	buy, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: buyIn}, TokenOut: tokenX})
	require.NoError(t, err)
	require.Equal(t, "7942941019685519127495", buy.TokenAmountOut.Amount.String())

	sellIn := bi("7942941019685519127494") // buy output minus the executor's 1 wei
	sell := func(s *PoolSimulator) string {
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: sellIn}, TokenOut: tokenY})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount.String()
	}

	// Stale reserves: the ~10x underquote seen in production.
	require.Equal(t, "10725731511188", sell(sim))

	// Chain reserves at block 56574352 (after the buy) give the chain's exact output.
	fresh := newTestSimulator(t, "1945741674322653757805", "124052347600320", 10, 100)
	require.Equal(t, "99425858078591", sell(fresh))
}
