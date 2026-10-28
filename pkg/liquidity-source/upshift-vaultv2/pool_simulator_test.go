package upshiftvaultv2

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	ausd   = "0x00000000efe302beaa2b3e6e1b18d08d69a9012a"
	earnLP = "0x103222f020e98bba0ad9809a011fdf8e6f067496"
	vault  = "0x36edbf0c834591bfdfcac0ef9605528c75c406aa"
)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

// earnExtra is monad earnAUSD at block 113871528. TotalAssets is net of the 80565555 collectable
// fees, Valuation (the previewDeposit denominator) is not.
func earnExtra() Extra {
	return Extra{
		Supply: u("17570622702819"), Valuation: u("18407691390778"), TotalAssets: u("18407610825223"),
		MaxDeposit: u("10000000000000000"), DepositCap: u("10000000000000000"),
	}
}

func newSim(t *testing.T, e Extra) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(e)
	require.NoError(t, err)
	a, l := e.Reserves()
	sim, err := NewPoolSimulator(entity.Pool{
		Address: vault, Exchange: DexType, Type: DexType,
		Reserves: []string{a.String(), l.String()},
		Tokens:   []*entity.PoolToken{{Address: ausd, Decimals: 6, Swappable: true}, {Address: earnLP, Decimals: 18, Swappable: true}},
		Extra:    string(extraBytes), BlockNumber: 113871528,
	})
	require.NoError(t, err)
	return sim
}

func quote(sim *PoolSimulator, amount string) (*pool.CalcAmountOutResult, error) {
	a, _ := new(big.Int).SetString(amount, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: ausd, Amount: a}, TokenOut: earnLP})
}

// Expected values are vault.previewDeposit at the same block.
func TestCalcAmountOut_MatchesPreviewDeposit(t *testing.T) {
	t.Parallel()
	sim := newSim(t, earnExtra())
	for amount, want := range map[string]string{
		"999999": "954525", "1000000": "954526", "123456789": "117842732",
		"1000000000000": "954526145066", "5000000000000": "4772630725334",
	} {
		res, err := quote(sim, amount)
		require.NoError(t, err, amount)
		assert.Equal(t, want, res.TokenAmountOut.Amount.String(), amount)
	}
}

func TestCalcAmountOut_Rejects(t *testing.T) {
	t.Parallel()
	paused, gated := earnExtra(), earnExtra()
	paused.DepositsPaused, gated.SenderGated = true, true
	empty := earnExtra()
	empty.Supply = new(uint256.Int)

	for name, tc := range map[string]struct {
		extra  Extra
		amount string
		err    error
	}{
		"paused":      {paused, "1000000", ErrDepositsPaused},
		"sender gate": {gated, "1000000", ErrSenderGated},
		"empty vault": {empty, "1000000", ErrNoSupply},
		"zero shares": {earnExtra(), "1", ErrInsufficientShr},
		"max deposit": {earnExtra(), "10000000000000001", ErrMaxDeposit},
	} {
		_, err := quote(newSim(t, tc.extra), tc.amount)
		assert.ErrorIs(t, err, tc.err, name)
	}
}

// Fork-verified: the cap is compared against TotalAssets (net of fees), allowing exactly
// DepositCap-TotalAssets and rejecting one more.
func TestCalcAmountOut_DepositCapUsesTotalAssets(t *testing.T) {
	t.Parallel()
	e := earnExtra()
	e.DepositCap = u("18407610825223")
	e.DepositCap.AddUint64(e.DepositCap, 200000000)
	sim := newSim(t, e)

	_, err := quote(sim, "200000000")
	assert.NoError(t, err)
	_, err = quote(sim, "200000001")
	assert.ErrorIs(t, err, ErrDepositCap)
}

// UpdateBalance must move the supply and both totals so a second deposit sees the first.
func TestUpdateBalance_Sequential(t *testing.T) {
	t.Parallel()
	e := earnExtra()
	e.DepositCap = u("18407610825223")
	e.DepositCap.AddUint64(e.DepositCap, 300000000)
	sim := newSim(t, e)
	clone := sim.CloneState()

	in := pool.TokenAmount{Token: ausd, Amount: big.NewInt(200000000)}
	res, err := quote(sim, "200000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})

	// 100000001 of headroom is left, so one more unit fails.
	_, err = quote(sim, "100000001")
	assert.ErrorIs(t, err, ErrDepositCap)
	res2, err := quote(sim, "100000000")
	require.NoError(t, err)
	// supply and valuation grew by shares and amount
	S, V := big.NewInt(17570622702819), big.NewInt(18407691390778)
	S.Add(S, res.TokenAmountOut.Amount)
	V.Add(V, big.NewInt(200000000))
	want := new(big.Int).Mul(big.NewInt(100000000), S)
	want.Div(want, V)
	assert.Equal(t, want.String(), res2.TokenAmountOut.Amount.String())

	// the clone taken before is untouched
	_, err = quote(clone.(*PoolSimulator), "300000000")
	assert.NoError(t, err)
}
