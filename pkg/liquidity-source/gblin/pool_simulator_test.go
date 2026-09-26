package gblin

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
	testWeth  = "0x4200000000000000000000000000000000000006"
	testVault = "0xc2181d975c05c8c724b334bced0764c0b86b1d53"
)

// testExtra is a state of the size of the live vault's: about 12.5 GBLIN outstanding, fees 5 + 5 bps, management
// fee 50 bps a year, last accrual 2,000 seconds before the block.
func testExtra() Extra {
	return Extra{
		Supply:           uint256.MustFromDecimal("12533838798982781550"),
		NavEth:           uint256.MustFromDecimal("495940133108446081"),
		LastAccrual:      1790318000,
		ManagementFeeBps: 50,
		ProtocolFeeBps:   5,
		StabilityFeeBps:  5,
		MinDeposit:       uint256.NewInt(0),
		NavReliable:      true,
		SequencerUp:      true,
		Timestamp:        1790320000,
	}
}

func newTestSimulator(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  testVault,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: testWeth, Decimals: 18, Swappable: true},
			{Address: testVault, Decimals: 18, Swappable: true},
		},
		Extra:       string(extraBytes),
		BlockNumber: 51764733,
	})
	require.NoError(t, err)
	return sim
}

// referenceMint is an independent math/big transcription of GBLIN._mintGBLIN.
func referenceMint(e Extra, amountIn *big.Int) (out, feeShares, accrued *big.Int) {
	bpsB, yearB := big.NewInt(10_000), big.NewInt(365*24*60*60)
	supply := e.Supply.ToBig()
	accrued = new(big.Int)
	if e.LastAccrual != 0 && e.Timestamp > e.LastAccrual && supply.Sign() > 0 && e.ManagementFeeBps != 0 {
		accrued.Mul(supply, new(big.Int).SetUint64(e.ManagementFeeBps))
		accrued.Mul(accrued, new(big.Int).SetUint64(e.Timestamp-e.LastAccrual))
		accrued.Div(accrued, new(big.Int).Mul(bpsB, yearB))
	}
	s := new(big.Int).Add(supply, accrued)
	s.Add(s, big.NewInt(1_000_000))
	a := new(big.Int).Add(e.NavEth.ToBig(), big.NewInt(1))
	pf := new(big.Int).Div(new(big.Int).Mul(amountIn, new(big.Int).SetUint64(e.ProtocolFeeBps)), bpsB)
	sf := new(big.Int).Div(new(big.Int).Mul(amountIn, new(big.Int).SetUint64(e.StabilityFeeBps)), bpsB)
	net := new(big.Int).Sub(amountIn, pf)
	net.Sub(net, sf)
	out = new(big.Int).Div(new(big.Int).Mul(net, s), a)
	feeShares = new(big.Int).Div(new(big.Int).Mul(pf, s), a)
	return out, feeShares, accrued
}

func quote(t *testing.T, sim *PoolSimulator, amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWeth, Amount: amountIn},
		TokenOut:      testVault,
	})
}

func TestCalcAmountOut_MatchesVaultFormula(t *testing.T) {
	sim := newTestSimulator(t, testExtra())
	for _, s := range []string{"1", "1000", "1000000000000000", "123456789012345678", "1000000000000000000",
		"5000000000000000000", "1000000000000000000000000"} {
		amountIn, _ := new(big.Int).SetString(s, 10)
		want, _, _ := referenceMint(testExtra(), amountIn)
		res, err := quote(t, sim, amountIn)
		if want.Sign() == 0 {
			assert.ErrorIs(t, err, ErrZeroAmountOut, s)
			continue
		}
		require.NoError(t, err, s)
		assert.Equal(t, want.String(), res.TokenAmountOut.Amount.String(), s)
		assert.Equal(t, defaultMintGas, res.Gas)
	}
}

func TestCalcAmountOut_AccruesManagementFee(t *testing.T) {
	withAccrual := testExtra()
	noAccrual := testExtra()
	noAccrual.LastAccrual = noAccrual.Timestamp
	amountIn := big.NewInt(1e18)

	resWith, err := quote(t, newTestSimulator(t, withAccrual), amountIn)
	require.NoError(t, err)
	resWithout, err := quote(t, newTestSimulator(t, noAccrual), amountIn)
	require.NoError(t, err)
	// the accrued fee shares dilute the NAV per share, so the same deposit mints more shares
	assert.Equal(t, 1, resWith.TokenAmountOut.Amount.Cmp(resWithout.TokenAmountOut.Amount))
}

func TestCalcAmountOut_Rejections(t *testing.T) {
	amountIn := big.NewInt(1e18)

	unreliable := testExtra()
	unreliable.NavReliable = false
	_, err := quote(t, newTestSimulator(t, unreliable), amountIn)
	assert.ErrorIs(t, err, ErrNavUnreliable)

	down := testExtra()
	down.SequencerUp = false
	_, err = quote(t, newTestSimulator(t, down), amountIn)
	assert.ErrorIs(t, err, ErrSequencerDown)

	minDeposit := testExtra()
	minDeposit.MinDeposit = uint256.NewInt(2e18)
	_, err = quote(t, newTestSimulator(t, minDeposit), amountIn)
	assert.ErrorIs(t, err, ErrDepositTooSmall)

	missing := testExtra()
	missing.NavEth = nil
	_, err = quote(t, newTestSimulator(t, missing), amountIn)
	assert.ErrorIs(t, err, ErrStateUnavailable)

	sim := newTestSimulator(t, testExtra())
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testVault, Amount: amountIn},
		TokenOut:      testWeth,
	})
	assert.ErrorIs(t, err, ErrUnsupportedSwap)

	_, err = quote(t, sim, big.NewInt(0))
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestUpdateBalance_AppliesMintOnce(t *testing.T) {
	e := testExtra()
	sim := newTestSimulator(t, e)
	amountIn := big.NewInt(1e18)

	res, err := quote(t, sim, amountIn)
	require.NoError(t, err)
	out, feeShares, accrued := referenceMint(e, amountIn)

	cloned := sim.CloneState()
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testWeth, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	wantSupply := new(big.Int).Add(e.Supply.ToBig(), accrued)
	wantSupply.Add(wantSupply, out)
	wantSupply.Add(wantSupply, feeShares)
	assert.Equal(t, wantSupply.String(), sim.extra.Supply.Dec())
	assert.Equal(t, new(big.Int).Add(e.NavEth.ToBig(), amountIn).String(), sim.extra.NavEth.Dec())
	assert.Equal(t, e.Timestamp, sim.extra.LastAccrual)

	// the next mint in the same block accrues nothing more and is priced on the updated state
	next := e
	next.Supply = uint256.MustFromBig(wantSupply)
	next.NavEth = uint256.MustFromBig(new(big.Int).Add(e.NavEth.ToBig(), amountIn))
	next.LastAccrual = e.Timestamp
	want2, _, _ := referenceMint(next, amountIn)
	res2, err := quote(t, sim, amountIn)
	require.NoError(t, err)
	assert.Equal(t, want2.String(), res2.TokenAmountOut.Amount.String())

	// the clone taken before the update still quotes on the original state
	res3, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWeth, Amount: amountIn},
		TokenOut:      testVault,
	})
	require.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount.String(), res3.TokenAmountOut.Amount.String())
}

func TestCanSwap(t *testing.T) {
	sim := newTestSimulator(t, testExtra())
	assert.Equal(t, []string{testVault}, sim.CanSwapFrom(testWeth))
	assert.Empty(t, sim.CanSwapFrom(testVault))
	assert.Equal(t, []string{testWeth}, sim.CanSwapTo(testVault))
	assert.Empty(t, sim.CanSwapTo(testWeth))
	assert.Equal(t, testVault, sim.GetApprovalAddress(testWeth, testVault))
}

func TestSequencerUp(t *testing.T) {
	now := uint64(1_000_000)
	b := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
	assert.True(t, sequencerUp(b(0), b(now-3601), b(now-3601), now))
	assert.False(t, sequencerUp(b(0), b(now-3600), b(now-3600), now), "grace period is inclusive")
	assert.False(t, sequencerUp(b(1), b(now-100000), b(now-100000), now), "reported down")
	assert.False(t, sequencerUp(b(0), b(0), b(now), now))
	assert.False(t, sequencerUp(b(0), b(now-5000), b(0), now))
	assert.False(t, sequencerUp(b(0), b(now+1), b(now+1), now))
	assert.False(t, sequencerUp(nil, b(now-5000), b(now-5000), now))
}
