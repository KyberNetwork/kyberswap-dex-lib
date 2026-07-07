package bouncetech

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	testUSDC = "0x0000000000000000000000000000000000000001"
	testLT   = "0x0000000000000000000000000000000000000002"
)

func TestCalcAmountOutRedeemUsesGrossAmountForReserveAndState(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")
	amountIn := mustBig("100000000000000000000") // 100 LT

	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testLT, Amount: amountIn},
		TokenOut:      testUSDC,
	})
	require.NoError(t, err)
	require.Equal(t, "97000000", res.TokenAmountOut.Amount.String())
	require.Equal(t, "3000000", res.Fee.Amount.String())

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testLT, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, "900000000", s.Info.Reserves[0].String())
	require.Equal(t, "900000000000000000000", s.Info.Reserves[1].String())
}

func TestCalcAmountOut(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000") // 1000 USDC base balance

	testutil.TestCalcAmountOut(t, s, map[int]map[int]map[string]string{
		0: { // USDC -> LT (mint)
			1: {
				"10000000": "10000000000000000000",    // exactly minTxSize: 10 USDC -> 10 LT
				"9999999":  ErrBelowMinAmount.Error(), // below minTxSize
			},
		},
		1: { // LT -> USDC (redeem)
			0: {
				"100000000000000000000":  "97000000",                     // 100 LT -> 97 USDC net (3% fee)
				"2000000000000000000000": ErrInsufficientBalance.Error(), // gross exceeds base balance
			},
		},
	})
}

func TestCloneState(t *testing.T) {
	s := newBounceTechTestSimulator(t, "1000000000")

	testutil.TestCloneState(t, s, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testLT, Amount: mustBig("100000000000000000000")},
		TokenOut:      testUSDC,
	}, nil)
}

func TestCalcAmountIn(t *testing.T) {
	// minTransactionSize is disabled here: it's a hard floor that testutil's power-of-10
	// exponent search isn't built to route around given the 18-decimal LT / 1e30 combined
	// mint-redeem scale, and it's already covered by the dedicated min-size tests below.
	testutil.TestCalcAmountIn(t, newBounceTechTestSimulator(t, "1000000000", "0"))
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
