package bouncetech

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testUSDC = "0x0000000000000000000000000000000000000001"
	testLT   = "0x0000000000000000000000000000000000000002"
)

func TestCalcAmtOutRedeemUsesGrossAmountForReserveAndState(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")
	amountIn := uint256.MustFromDecimal("100000000000000000000") // 100 LT

	res, err := s.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testLT, Amount: *amountIn},
		TokenOut:      testUSDC,
	})
	require.NoError(t, err)
	require.Equal(t, "97000000", res.TokenAmountOut.Amount.Dec())
	fee := res.SwapInfo.(SwapInfo).Fee
	require.Equal(t, "3000000", fee.Dec())

	s.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  pool.TokenAmt{Token: testLT, Amount: *amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, "900000000", s.Info.Reserves[0].String())
	require.Equal(t, "900000000000000000000", s.Info.Reserves[1].String())
}

func TestCalcAmtOut(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000") // 1000 USDC base balance

	cases := []struct {
		name    string
		tokenIn string
		amtIn   string
		wantOut string
		wantErr error
	}{
		{"mint-at-min", testUSDC, "10000000", "10000000000000000000", nil},
		{"mint-below-min", testUSDC, "9999999", "", ErrBelowMinAmount},
		{"redeem-net-of-fee", testLT, "100000000000000000000", "97000000", nil},
		{"redeem-gross-exceeds-balance", testLT, "2000000000000000000000", "", ErrInsufficientBalance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := s.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tc.tokenIn, Amount: *uint256.MustFromDecimal(tc.amtIn)},
				TokenOut:      otherToken(tc.tokenIn),
			})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantOut, res.TokenAmountOut.Amount.Dec())
		})
	}
}

// TestCalcAmtOutMatchesLegacyGolden pins the native quote to values the legacy big.Int CalcAmountOut produced
// before the port. It checks out, fee, and gas, so a change to the shared quote core fails here.
func TestCalcAmtOutMatchesLegacyGolden(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")

	cases := []struct {
		name             string
		tokenIn, amtIn   string
		wantOut, wantFee string
		wantGas          int64
		wantMint         bool
	}{
		{"redeem-100-lt", testLT, "100000000000000000000", "97000000", "3000000", redeemGas, false},
		{"mint-10-usdc", testUSDC, "10000000", "10000000000000000000", "0", mintGas, true},
		{"mint-1e18-usdc", testUSDC, "1000000000000000000", "1000000000000000000000000000000", "0", mintGas, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := s.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tc.tokenIn, Amount: *uint256.MustFromDecimal(tc.amtIn)},
				TokenOut:      otherToken(tc.tokenIn),
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantOut, res.TokenAmountOut.Amount.Dec())
			require.Equal(t, tc.wantGas, res.Gas)
			require.Equal(t, SwapInfo{IsMint: tc.wantMint, Fee: *uint256.MustFromDecimal(tc.wantFee)}, res.SwapInfo)
		})
	}
}

func otherToken(token string) string {
	if token == testUSDC {
		return testLT
	}
	return testUSDC
}

// TestCloneStateIsIPoolSim checks the clone satisfies IPoolSim, the uint256 interface the factory returns.
func TestCloneStateIsIPoolSim(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")

	var clone pool.IPoolSim = s.CloneState()
	require.NotNil(t, clone)
	require.Equal(t, DexType, clone.GetType())
}

// TestCloneStateDeepCopies checks that UpdateState on the clone leaves the original reserves unchanged.
func TestCloneStateDeepCopies(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")
	origReserves := s.GetReserves()

	clone := s.CloneState().(*PoolSimulator)
	clone.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  pool.TokenAmt{Token: testLT, Amount: *uint256.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmt{Token: testUSDC, Amount: *uint256.NewInt(970000)},
		SwapInfo:       SwapInfo{IsMint: false, Fee: *uint256.NewInt(30000)},
	})

	require.Equal(t, origReserves, s.GetReserves(), "UpdateState on the clone changed the original reserves")
}

// TestCalcAmountInCoversCalcAmtOut checks CalcAmountIn's amountIn yields at least the requested amountOut when
// quoted forward. testutil.TestCalcAmountIn cannot run here: it takes IPoolSimulator, which this type no longer has.
func TestCalcAmountInCoversCalcAmtOut(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000", "0")

	for _, tc := range []struct{ tokenIn, tokenOut, amountOut string }{
		{testUSDC, testLT, "10000000000000000000"},
		{testLT, testUSDC, "97000000"},
	} {
		in, err := s.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: mustBig(tc.amountOut)},
			TokenIn:        tc.tokenIn,
		})
		require.NoError(t, err)

		out, err := s.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tc.tokenIn, Amount: *uint256.MustFromBig(in.TokenAmountIn.Amount)},
			TokenOut:      tc.tokenOut,
		})
		require.NoError(t, err)
		require.True(t, out.TokenAmountOut.Amount.Cmp(uint256.MustFromBig(mustBig(tc.amountOut))) >= 0,
			"amountIn %s gives %s out, want at least %s", in.TokenAmountIn.Amount, out.TokenAmountOut.Amount.Dec(), tc.amountOut)
	}
}

// TestCalcAmountInRedeemOnFixturePoolHitsInsufficientBalance documents that the fetched
// hyperevm fixture pool's real USDC reserve is smaller than its own minTransactionSize,
// so any redeem large enough to clear the minimum necessarily exceeds the available balance.
func TestCalcAmountInRedeemOnFixturePoolHitsInsufficientBalance(t *testing.T) {
	s := newBounceTechFixtureSimulator(t)

	_, err := s.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testFixtureUSDC, Amount: mustBig("10000000")},
		TokenIn:        testFixtureLT,
	})
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestCalcAmountInMintRejectsBelowMinTransactionSize(t *testing.T) {
	s := newBounceTechFixtureSimulator(t)

	_, err := s.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testFixtureLT, Amount: big.NewInt(1)},
		TokenIn:        testFixtureUSDC,
	})
	require.ErrorIs(t, err, ErrBelowMinAmount)
}

const (
	testFixtureUSDC = "0xb88339cb7199b77e23db6e890353e22632ba630f"
	testFixtureLT   = "0xdfde51b58e6c143ef41659f66bde4614a4a27786"
)

// newBounceTechFixtureSimulator builds a simulator from a real hyperevm pool
// (fetched via the router API) to exercise CalcAmountIn against realistic state.
func newBounceTechFixtureSimulator(t *testing.T) *PoolSimulator {
	t.Helper()

	s, err := NewPoolSimulator(entity.Pool{
		Address:  testFixtureLT,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{
			"9296560",
			"10000000000000000000",
		},
		Tokens: []*entity.PoolToken{
			{Address: testFixtureUSDC, Swappable: true},
			{Address: testFixtureLT, Swappable: true},
		},
		Extra: `{"exchangeRate":"929656000000000000","redemptionFee":"3000000000000000","targetLeverage":"2000000000000000000","minTransactionSize":"10000000","mintPaused":false}`,
	})
	require.NoError(t, err)
	return s
}

func newBounceTechTestSimulator(t *testing.T, baseBalance string, minTxSizeOpt ...string) *PoolSimulator {
	t.Helper()

	minTxSize := uint256.NewInt(10_000_000)
	if len(minTxSizeOpt) > 0 {
		minTxSize = uint256.MustFromDecimal(minTxSizeOpt[0])
	}

	extraBytes, err := json.Marshal(Extra{
		ExchangeRate:       uint256.NewInt(1e18),
		RedemptionFee:      uint256.NewInt(1e16), // 1%
		TargetLeverage:     uint256.NewInt(3e18),
		MinTransactionSize: minTxSize,
	})
	require.NoError(t, err)

	s, err := NewPoolSimulator(entity.Pool{
		Address:  testLT,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{
			baseBalance,
			"1000000000000000000000",
		},
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testLT, Swappable: true},
		},
		Extra: string(extraBytes),
	})
	require.NoError(t, err)
	return s
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big int")
	}
	return v
}
