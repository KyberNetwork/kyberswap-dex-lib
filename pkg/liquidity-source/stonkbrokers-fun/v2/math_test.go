package stonkbrokersfunv2

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func uDec(t *testing.T, s string) *uint256.Int {
	t.Helper()
	v, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return v
}

// TestCalcBuyAmountOut_LiveVerifiedWethLaunch176 ports the reference fixture
// captured live in output/explorer.md: WETH V2 pad
// (0xFCd61B25BbF3AbD6cf0070D6328E351cc30EEC9f), on-chain launch id 176,
// buying 0.01 ETH at block 46325622. Both getLaunch(176) and
// quoteBuy(pad,176,0.01 ETH) were called directly against the chain's
// canonical RPC; this test asserts the ported formula reproduces the lens's
// own quoteBuy output bit-for-bit.
func TestCalcBuyAmountOut_LiveVerifiedWethLaunch176(t *testing.T) {
	vQuote := uDec(t, "410349618506110198")
	vToken := uDec(t, "999010726104174939447103989")
	quoteIn := uDec(t, "10000000000000000") // 0.01 ETH
	const taxBps = 100                      // openEnded, past window -> flat postTaxBps

	tokensOut, tax, newVQuote, newVToken, err := CalcBuyAmountOut(quoteIn, vQuote, vToken, taxBps)
	require.NoError(t, err)

	require.Equal(t, "23534122942428164868379888", tokensOut.Dec())
	require.Equal(t, "100000000000000", tax.Dec()) // 0.0001 ETH == 1% of 0.01 ETH

	wantNewVQuote := uDec(t, "420249618506110198") // vQuote + (quoteIn - tax)
	wantNewVToken := new(uint256.Int).Sub(vToken, tokensOut)
	require.True(t, newVQuote.Eq(wantNewVQuote))
	require.True(t, newVToken.Eq(wantNewVToken))
}

func TestCalcBuyAmountOut_ZeroTradeRejected(t *testing.T) {
	vQuote := uDec(t, "1000000000000000000")
	vToken := uDec(t, "1000000000000000000000")

	_, _, _, _, err := CalcBuyAmountOut(uint256.NewInt(0), vQuote, vToken, 100)
	require.ErrorIs(t, err, ErrZeroTrade)

	// A quoteIn small enough that the tax rounds it to zero net is also a
	// zero-trade, not a zero-output-but-otherwise-fine quote.
	_, _, _, _, err = CalcBuyAmountOut(uint256.NewInt(1), vQuote, vToken, 10_000)
	require.ErrorIs(t, err, ErrZeroTrade)
}

// TestCurrentTaxBps ports StonkSafeLaunchpadV2.currentTaxBps's four branches:
// buffer (flat snipe-shield), linear decay, floor at 0, and the openEnded
// flat post-tax floor. BufferTaxBps=9999 and MaxBuyPpm-adjacent constants
// are the live values read from the WETH pad (BUFFER_TAX_BPS()).
func TestCurrentTaxBps(t *testing.T) {
	base := StaticExtra{
		BufferTaxBps:      9999,
		StartTaxBps:       5000,
		DecayPerMinuteBps: 500,
		BufferSecs:        600,
		StartTime:         1_000_000,
	}

	tests := []struct {
		name string
		se   StaticExtra
		now  uint64
		want uint16
	}{
		{"inside buffer", base, 1_000_000 + 300, 9999},
		{"exactly at buffer boundary", base, 1_000_000 + 600, 5000},
		{"partway through decay", base, 1_000_000 + 600 + 3*60, 5000 - 3*500}, // 3500
		{"decay reaches zero, not open-ended", base, 1_000_000 + 600 + 10*60, 0},
		{
			name: "decay reaches zero, open-ended falls to postTaxBps",
			se: StaticExtra{
				BufferTaxBps: 9999, StartTaxBps: 5000, DecayPerMinuteBps: 500,
				BufferSecs: 600, StartTime: 1_000_000, OpenEnded: true, PostTaxBps: 100,
			},
			now:  1_000_000 + 600 + 10*60,
			want: 100,
		},
		{
			name: "zero window+buffer, openEnded -> immediately flat post-tax (matches live launch 176)",
			se: StaticExtra{
				BufferTaxBps: 9999, StartTaxBps: 0, DecayPerMinuteBps: 0,
				BufferSecs: 0, StartTime: 1_787_691_927, OpenEnded: true, PostTaxBps: 100,
			},
			now:  1_787_691_927 + 28_572,
			want: 100,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, CurrentTaxBps(tc.now, tc.se))
		})
	}
}

// TestMcapUsd8FromReserves_GraduationBoundary exercises the mcap comparison
// that decides whether a buy closes the curve.
func TestMcapUsd8FromReserves_GraduationBoundary(t *testing.T) {
	loadedSupply := uDec(t, "1000000000000000000000000000") // 1e9 tokens, 18 decimals
	vToken := uDec(t, "999010726104174939447103989")
	vQuote := uDec(t, "420249618506110198")
	const quoteDecimals = 18
	const ethUsd8 = 245_794_157_469 // $2457.94157469, live ETH/USD feed value

	mcap := McapUsd8FromReserves(vQuote, vToken, loadedSupply, ethUsd8, quoteDecimals)
	require.False(t, mcap.IsZero())

	zeroVToken := uint256.NewInt(0)
	require.True(t, McapUsd8FromReserves(vQuote, zeroVToken, loadedSupply, ethUsd8, quoteDecimals).IsZero())
}

// TestDirectFeedUsd8_StalePriceRejected mirrors SafeLaunchTwapLib._feedUsd8's
// fail-closed staleness gate (48h) -- this is the ErrStalePrice enforcement
// path the coordinator asked to be made real, not just declared.
func TestDirectFeedUsd8_StalePriceRejected(t *testing.T) {
	fresh := OracleReading{Answer: uint256.NewInt(245_794_157_469), Decimals: 8, UpdatedAt: 1_787_719_697, Ok: true}

	got, err := DirectFeedUsd8(fresh, 1_787_719_697+3600) // 1h later, fresh
	require.NoError(t, err)
	require.Equal(t, uint64(245_794_157_469), got)

	_, err = DirectFeedUsd8(fresh, 1_787_719_697+48*3600+1) // 48h+1s later, stale
	require.ErrorIs(t, err, ErrStalePrice)

	notOk := OracleReading{Ok: false}
	_, err = DirectFeedUsd8(notOk, 1_787_719_697)
	require.ErrorIs(t, err, ErrBadOracleAnswer)

	badAnswer := OracleReading{Answer: uint256.NewInt(0), Decimals: 8, UpdatedAt: 1_787_719_697, Ok: true}
	_, err = DirectFeedUsd8(badAnswer, 1_787_719_697)
	require.ErrorIs(t, err, ErrBadOracleAnswer)
}

// TestDirectFeedUsd8_RescalesNonE8Decimals exercises the decimals != 8
// rescale branch (feed-mode lanes are not guaranteed to publish at 1e8).
func TestDirectFeedUsd8_RescalesNonE8Decimals(t *testing.T) {
	// A realistic 1e18-scaled feed answer worth $2457.94157469 (same mark as
	// the live 1e8 ETH/USD feed used elsewhere in this file) must rescale to
	// the identical 1e8 usd8 value: mulDiv(x, 1e8, 1e18) == x / 1e10.
	r := OracleReading{Answer: uDec(t, "2457941574690000000000"), Decimals: 18, UpdatedAt: 1000, Ok: true}
	got, err := DirectFeedUsd8(r, 1000)
	require.NoError(t, err)
	require.Equal(t, uint64(245_794_157_469), got)
}

// TestTwapQuoteUsd8_LiveVerifiedUsdgLaunch8 ports the second reference
// fixture captured live in output/math.md: USDG V2 pad
// (0xd4F20033586977A2511f4A2DB4aF7C79a340D70a), TWAP pool
// 0x52e65b17fb6e5ba00ed806f37afcd2daa50271ca (token0=WETH, token1=USDG,
// so quoteIsToken0=false), observe([1800,0]) and the shared ETH/USD feed
// were both called directly against the chain's canonical RPC. The result
// is cross-checked against the pad's own mcapUsd8(8) getter in
// output/math.md (implied quoteUsd8 ~= 99786960, matching this test's exact
// integer result).
func TestTwapQuoteUsd8_LiveVerifiedUsdgLaunch8(t *testing.T) {
	reading := TwapReading{
		TickCumulativeOld: -942871677686,
		TickCumulativeNow: -943228487508,
		EthUsd: OracleReading{
			Answer:    uint256.NewInt(245_794_157_469),
			Decimals:  8,
			UpdatedAt: 1_787_719_697,
			Ok:        true,
		},
		Ok: true,
	}

	got, err := TwapQuoteUsd8(reading, 1800, false, 6, 1_787_720_500)
	require.NoError(t, err)
	require.Equal(t, uint64(99_786_960), got)
}

func TestTwapQuoteUsd8_ObserveFailurePropagates(t *testing.T) {
	_, err := TwapQuoteUsd8(TwapReading{Ok: false}, 1800, false, 6, 1000)
	require.ErrorIs(t, err, ErrBadOracleAnswer)
}
