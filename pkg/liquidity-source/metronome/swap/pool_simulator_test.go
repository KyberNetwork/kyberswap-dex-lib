package metronomeswap

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
	msETH = "0x64351fC9810aDAd17A690E4e1717Df5e7e085160"
	msUSD = "0xab5eB14c09D416F0aC63661E57EDB7AEcDb9BEfA"
	msBTC = "0xB93f48D3eA42a25f367fAde092A6Bb56DAB5F7cB"
)

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	t.Helper()

	extra := Extra{
		SwapActive:   true,
		FeeProvider:  "0x6b53C16B94c1502C661140073ed522aC7Dbc5E5E",
		MasterOracle: "0x80704Acdf97723963263c78F861F091ad04F46E2",
		Tokens: map[string]TokenState{
			msETH: {
				IsActive:       true,
				MaxTotalSupply: uint256.MustFromDecimal("50000000000000000000000"),
				TotalSupply:    uint256.MustFromDecimal("10762835413342885949867"),
				PriceInUsd:     fixturePriceETH,
			},
			msUSD: {
				IsActive:       true,
				MaxTotalSupply: uint256.MustFromDecimal("100000000000000000000000000"),
				TotalSupply:    uint256.MustFromDecimal("27701309817703735650985241"),
				PriceInUsd:     fixturePriceUSD,
			},
			msBTC: {
				IsActive:       false, // deliberately inactive, for the gating test
				MaxTotalSupply: uint256.MustFromDecimal("1000000000000000000000"),
				TotalSupply:    uint256.MustFromDecimal("1712118682323450"),
				PriceInUsd:     uint256.MustFromDecimal("100000000000000000000000"),
			},
		},
		SwapFeesBps: map[string]*uint256.Int{
			msETH + "-" + msUSD: fixtureFeeBps,
		},
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	staticExtra := StaticExtra{PoolRegistry: "0x11eaD85C679eAF528c9C1FE094bF538Db880048A"}
	staticExtraBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	p := entity.Pool{
		Address:  "0x3364f53cB866762Aef66DeEF2a6b1a17C1F17f46",
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: msETH, Decimals: 18, Swappable: true},
			{Address: msUSD, Decimals: 18, Swappable: true},
			{Address: msBTC, Decimals: 18, Swappable: true},
		},
		Reserves:    []string{"39237164586657114050133", "72298690182296264349014759", "999998287881317677"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOut_MatchesRealSwap(t *testing.T) {
	sim := newTestPoolSimulator(t)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenOut:      msUSD,
	})
	require.NoError(t, err)

	assert.Equal(t, fixtureNetOut.String(), result.TokenAmountOut.Amount.String())
	assert.Equal(t, fixtureFee.String(), result.Fee.Amount.String())
	assert.Equal(t, msUSD, result.Fee.Token)
}

func TestCalcAmountOut_SameToken(t *testing.T) {
	sim := newTestPoolSimulator(t)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: big.NewInt(1)},
		TokenOut:      msETH,
	})
	assert.ErrorIs(t, err, ErrSameToken)
}

func TestCalcAmountOut_UnknownToken(t *testing.T) {
	sim := newTestPoolSimulator(t)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x000000000000000000000000000000deadbeef", Amount: big.NewInt(1)},
		TokenOut:      msUSD,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_SwapInactive(t *testing.T) {
	sim := newTestPoolSimulator(t)
	sim.Extra.SwapActive = false
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenOut:      msUSD,
	})
	assert.ErrorIs(t, err, ErrSwapInactive)
}

func TestCalcAmountOut_TokenInactive(t *testing.T) {
	sim := newTestPoolSimulator(t)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: big.NewInt(1e6)},
		TokenOut:      msBTC, // inactive in the fixture
	})
	assert.ErrorIs(t, err, ErrTokenInactive)
}

func TestCalcAmountOut_ZeroAmountIn(t *testing.T) {
	sim := newTestPoolSimulator(t)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: big.NewInt(0)},
		TokenOut:      msUSD,
	})
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_ExceedsMaxTotalSupply(t *testing.T) {
	sim := newTestPoolSimulator(t)
	// msUSD headroom is ~72.3M; ask for way more msETH in than that could ever cover isn't
	// realistic, so instead shrink the cap directly to force the mint-cap error path.
	state := sim.Extra.Tokens[msUSD]
	state.MaxTotalSupply = new(uint256.Int).Set(state.TotalSupply) // zero headroom
	sim.Extra.Tokens[msUSD] = state

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenOut:      msUSD,
	})
	assert.ErrorIs(t, err, ErrExceedsMaxTotalSupply)
}

// TestCalcAmountOut_CapCheckedAgainstGross_NotNet locks in the fix confirmed against a real
// vnet swap(): Pool.swap() mints the fee to the fee collector AND the net amount to the
// caller, so the maxTotalSupply check is effectively against gross (net+fee), not just the
// net amount the trader receives. Headroom here is set strictly between net and gross — a
// net-only check would (incorrectly) allow this swap.
func TestCalcAmountOut_CapCheckedAgainstGross_NotNet(t *testing.T) {
	sim := newTestPoolSimulator(t)

	headroom := new(uint256.Int).Add(fixtureNetOut, uint256.NewInt(1e18)) // > net, < net+fee (fee ~4.9e18)
	state := sim.Extra.Tokens[msUSD]
	state.MaxTotalSupply = new(uint256.Int).Add(state.TotalSupply, headroom)
	sim.Extra.Tokens[msUSD] = state

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenOut:      msUSD,
	})
	assert.ErrorIs(t, err, ErrExceedsMaxTotalSupply)
}

func TestCalcAmountOut_MissingFeeDefaultsToZero(t *testing.T) {
	sim := newTestPoolSimulator(t)
	// no msUSD->msETH entry in SwapFeesBps fixture — must not panic, fee should be zero.
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: msUSD, Amount: big.NewInt(1_000000000000000000)},
		TokenOut:      msETH,
	})
	require.NoError(t, err)
	assert.Equal(t, "0", result.Fee.Amount.String())
}

func TestUpdateBalance(t *testing.T) {
	sim := newTestPoolSimulator(t)
	indexIn, indexOut := sim.GetTokenIndex(msETH), sim.GetTokenIndex(msUSD)

	beforeIn := new(big.Int).Set(sim.Info.Reserves[indexIn])
	beforeOut := new(big.Int).Set(sim.Info.Reserves[indexOut])

	// Gross = net + fee is what actually gets minted on-chain (Pool.swap() mints the fee to
	// the fee collector AND the net amount to the caller — two separate mints, both counted
	// against totalSupply/maxTotalSupply). See pool_simulator.go's UpdateBalance doc comment.
	grossAmountOut := new(uint256.Int).Add(fixtureNetOut, fixtureFee)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: msETH, Amount: fixtureAmountIn.ToBig()},
		TokenAmountOut: pool.TokenAmount{Token: msUSD, Amount: fixtureNetOut.ToBig()},
		Fee:            pool.TokenAmount{Token: msUSD, Amount: fixtureFee.ToBig()},
	})

	assert.Equal(t, fixtureAmountIn.String(), new(big.Int).Sub(sim.Info.Reserves[indexIn], beforeIn).String())
	assert.Equal(t, grossAmountOut.String(), new(big.Int).Sub(beforeOut, sim.Info.Reserves[indexOut]).String())

	assert.Equal(t, new(uint256.Int).Sub(uint256.MustFromDecimal("10762835413342885949867"), fixtureAmountIn).String(),
		sim.Extra.Tokens[msETH].TotalSupply.String())
	assert.Equal(t, new(uint256.Int).Add(uint256.MustFromDecimal("27701309817703735650985241"), grossAmountOut).String(),
		sim.Extra.Tokens[msUSD].TotalSupply.String())
}

func TestCloneState_DeepCopiesMutableFields(t *testing.T) {
	sim := newTestPoolSimulator(t)
	cloned := sim.CloneState().(*PoolSimulator)

	cloned.Extra.Tokens[msETH] = TokenState{
		IsActive:       true,
		MaxTotalSupply: uint256.NewInt(1),
		TotalSupply:    uint256.NewInt(1),
		PriceInUsd:     uint256.NewInt(1),
	}
	cloned.Info.Reserves[0].SetInt64(1)
	cloned.Extra.SwapFeesBps[msETH+"-"+msUSD].SetUint64(1)

	assert.NotEqual(t, sim.Extra.Tokens[msETH].PriceInUsd.String(), cloned.Extra.Tokens[msETH].PriceInUsd.String())
	assert.NotEqual(t, sim.Info.Reserves[0].String(), cloned.Info.Reserves[0].String())
	assert.NotEqual(t, sim.Extra.SwapFeesBps[msETH+"-"+msUSD].String(), cloned.Extra.SwapFeesBps[msETH+"-"+msUSD].String())
}

func TestCanSwapTo_CompleteGraph(t *testing.T) {
	sim := newTestPoolSimulator(t)
	to := sim.CanSwapTo(msETH)
	assert.ElementsMatch(t, []string{msUSD, msBTC}, to)
	assert.ElementsMatch(t, sim.CanSwapFrom(msETH), to)
}
