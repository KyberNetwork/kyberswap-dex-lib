package slyngfun

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// SLYNG's live curve on Robinhood Chain mainnet as the tracker read it at block 64859506
// (timestamp 1789597038, well past the opening window). Every expected figure below that the
// launchpad can answer for itself was read from it at the same block with quoteToTokens and
// tokensToQuote; the rest follow from those by the fee arithmetic in _buy and sell.
const slyngPoolJSON = `{
	"address": "0x066ffa6aaf8b54c6094d1ed1be818afa5e0f290e",
	"exchange": "slyng-fun",
	"type": "slyng-fun",
	"timestamp": 1789597038,
	"reserves": ["100225464022454202", "937367910787696446494526649"],
	"tokens": [
		{"address": "0x0bd7d308f8e1639fab988df18a8011f41eacad73", "symbol": "WETH", "decimals": 18, "swappable": true},
		{"address": "0x066ffa6aaf8b54c6094d1ed1be818afa5e0f290e", "symbol": "SLYNG", "decimals": 18, "swappable": true}
	],
	"extra": "{\"quoteReserve\":\"100225464022454202\",\"tokenReserve\":\"937367910787696446494526649\",\"graduated\":false}",
	"staticExtra": "{\"launchpad\":\"0xce0abc33ec4264377045ae17f9b887db33cc95b4\",\"isNativeQuote\":true,\"graduationTarget\":\"5000000000000000000\",\"virtualQuote\":\"1500000000000000000\",\"createdAt\":1789435818,\"tradeFeeBps\":100,\"snipeBps\":5000,\"snipeWindowSeconds\":30}",
	"blockNumber": 64859506
}`

const (
	slyngQuote     = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	slyngToken     = "0x066ffa6aaf8b54c6094d1ed1be818afa5e0f290e"
	slyngLaunchpad = "0xce0abc33ec4264377045ae17f9b887db33cc95b4"
	slyngCreatedAt = 1789435818
	fixtureNow     = 1789597038
)

func newSlyngSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(slyngPoolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func atTime(t *testing.T, now uint64) {
	t.Helper()
	prev := nowUnix
	nowUnix = func() uint64 { return now }
	t.Cleanup(func() { nowUnix = prev })
}

func bigStr(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad number " + s)
	}
	return v
}

func buy(sim *PoolSimulator, amountIn string) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: slyngQuote, Amount: bigStr(amountIn)},
		TokenOut:      slyngToken,
	})
}

func sell(sim *PoolSimulator, tokensIn string) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: slyngToken, Amount: bigStr(tokensIn)},
		TokenOut:      slyngQuote,
	})
}

func TestBuy_MatchesTheLaunchpad(t *testing.T) {
	atTime(t, fixtureNow)

	cases := []struct{ amountIn, tokensOut, fee, newQuote, newToken string }{
		// quoteToTokens(0.0099 ETH) at block 64859506
		{"10000000000000000", "5763490190146312875989179", "100000000000000",
			"110125464022454202", "931604420597550133618537470"},
		// quoteToTokens(0.99 ETH)
		{"1000000000000000000", "358267743317874688977668731", "10000000000000000",
			"1090225464022454202", "579100167469821757516857918"},
	}
	for _, c := range cases {
		sim := newSlyngSimulator(t)
		res, err := buy(sim, c.amountIn)
		require.NoError(t, err)
		assert.Equal(t, c.tokensOut, res.TokenAmountOut.Amount.String())
		assert.Equal(t, slyngToken, res.TokenAmountOut.Token)
		assert.Equal(t, c.fee, res.Fee.Amount.String(), "the fee is on the quote leg")
		assert.Equal(t, slyngQuote, res.Fee.Token)
		assert.Nil(t, res.RemainingTokenAmountIn, "a buy spends every wei")
		assert.EqualValues(t, defaultGas.BuyNative, res.Gas)

		si, ok := res.SwapInfo.(*SwapInfo)
		require.True(t, ok)
		assert.True(t, si.IsBuy)
		assert.True(t, si.IsNativeQuote)
		assert.Equal(t, slyngLaunchpad, si.Launchpad)
		assert.Equal(t, slyngToken, si.Token)
		assert.False(t, si.NewGraduated)
		assert.Equal(t, c.newQuote, si.NewQuoteReserve.Dec(), "the curve keeps everything but the fee")
		assert.Equal(t, c.newToken, si.NewTokenReserve.Dec())
	}
}

func TestSell_MatchesTheLaunchpad(t *testing.T) {
	atTime(t, fixtureNow)

	cases := []struct{ tokensIn, quoteOut, fee, newQuote, newToken string }{
		// tokensToQuote(1e24) = 1705328417165474 gross
		{"1000000000000000000000000", "1688275132993820", "17053284171654",
			"98520135605288728", "938367910787696446494526649"},
		// tokensToQuote(5e25) = 81034913457225683 gross
		{"50000000000000000000000000", "80224564322653427", "810349134572256",
			"19190550565228519", "987367910787696446494526649"},
		// the whole curve back: tokensToQuote rounds up past the reserve (800112732011227101) and
		// the launchpad clamps to what it holds rather than revert on the final exit
		{"937367910787696446494526649", "99223209382229660", "1002254640224542",
			"0", "1874735821575392892989053298"},
	}
	for _, c := range cases {
		sim := newSlyngSimulator(t)
		res, err := sell(sim, c.tokensIn)
		require.NoError(t, err)
		assert.Equal(t, c.quoteOut, res.TokenAmountOut.Amount.String())
		assert.Equal(t, slyngQuote, res.TokenAmountOut.Token)
		assert.Equal(t, c.fee, res.Fee.Amount.String())
		assert.Equal(t, slyngQuote, res.Fee.Token)
		assert.EqualValues(t, defaultGas.SellNative, res.Gas)

		si, ok := res.SwapInfo.(*SwapInfo)
		require.True(t, ok)
		assert.False(t, si.IsBuy)
		assert.Equal(t, c.newQuote, si.NewQuoteReserve.Dec())
		assert.Equal(t, c.newToken, si.NewTokenReserve.Dec())
	}
}

func TestBuy_OpeningSurcharge(t *testing.T) {
	// ten seconds into the curve's life: half of the input is withheld, on top of the fee
	atTime(t, slyngCreatedAt+10)
	sim := newSlyngSimulator(t)
	res, err := buy(sim, "10000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "2861522582383914745327828", res.TokenAmountOut.Amount.String())
	assert.Equal(t, "5100000000000000", res.Fee.Amount.String(), "fee plus surcharge")
	si := res.SwapInfo.(*SwapInfo)
	assert.Equal(t, "110125464022454202", si.NewQuoteReserve.Dec(),
		"the surcharge stays in the reserve and raises the price for the next buyer")
	assert.Equal(t, "934506388205312531749198821", si.NewTokenReserve.Dec())

	// the surcharge is flat across the window and gone the second it ends
	atTime(t, slyngCreatedAt+29)
	edge, err := buy(newSlyngSimulator(t), "10000000000000000")
	require.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount, edge.TokenAmountOut.Amount)

	atTime(t, slyngCreatedAt+30)
	after, err := buy(newSlyngSimulator(t), "10000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "5763490190146312875989179", after.TokenAmountOut.Amount.String())

	// a clock behind the chain is treated as inside the window
	atTime(t, slyngCreatedAt-5)
	early, err := buy(newSlyngSimulator(t), "10000000000000000")
	require.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount, early.TokenAmountOut.Amount)

	// a sell never pays it
	sold, err := sell(newSlyngSimulator(t), "1000000000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "1688275132993820", sold.TokenAmountOut.Amount.String())
}

func TestBuy_ThatGraduatesTheCurve(t *testing.T) {
	atTime(t, fixtureNow)
	sim := newSlyngSimulator(t)

	// quoteToTokens(4.95 ETH): the net lifts the reserve to 5.05 ETH, past the 5 ETH target
	res, err := buy(sim, "5000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "708368159826626628679478316", res.TokenAmountOut.Amount.String(),
		"the graduating buy is priced like any other")
	si := res.SwapInfo.(*SwapInfo)
	assert.True(t, si.NewGraduated)
	assert.Equal(t, "5050225464022454202", si.NewQuoteReserve.Dec())
	assert.EqualValues(t, defaultGas.BuyNative+defaultGas.Graduation, res.Gas)

	// nothing trades on the curve after it
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: si})
	_, err = buy(sim, "1000000000000000")
	assert.ErrorIs(t, err, ErrGraduated)
	_, err = sell(sim, "1000000000000000000000")
	assert.ErrorIs(t, err, ErrGraduated)
}

func TestGraduatedCurve_DoesNotTrade(t *testing.T) {
	atTime(t, fixtureNow)
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(slyngPoolJSON), &ep))
	ep.Extra = `{"quoteReserve":"0","tokenReserve":"0","graduated":true}`
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	_, err = buy(sim, "1000000000000000")
	assert.ErrorIs(t, err, ErrGraduated)
	_, err = sell(sim, "1000000000000000000000")
	assert.ErrorIs(t, err, ErrGraduated)
}

func TestRejects(t *testing.T) {
	atTime(t, fixtureNow)
	sim := newSlyngSimulator(t)

	_, err := buy(sim, "0")
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = sell(sim, "-1")
	assert.ErrorIs(t, err, ErrInvalidAmount)

	// the fee and the surcharge truncate to nothing on a wei, so even one wei buys dust; what
	// the launchpad refuses is a sell into a curve nobody has bought from, which holds nothing
	atTime(t, slyngCreatedAt+1)
	dust, err := buy(sim, "1")
	require.NoError(t, err)
	assert.Positive(t, dust.TokenAmountOut.Amount.Sign())

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(slyngPoolJSON), &ep))
	ep.Extra = `{"quoteReserve":"0","tokenReserve":"1000000000000000000000000000","graduated":false}`
	empty, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	_, err = sell(empty, "1000000000000000000000")
	assert.ErrorIs(t, err, ErrZeroAmount)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: slyngQuote, Amount: big.NewInt(1e15)},
		TokenOut:      slyngQuote,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x0000000000000000000000000000000000000001", Amount: big.NewInt(1e15)},
		TokenOut:      slyngToken,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestUpdateBalance_AndCloneState(t *testing.T) {
	atTime(t, fixtureNow)
	sim := newSlyngSimulator(t)
	clone := sim.CloneState().(*PoolSimulator)

	first, err := buy(sim, "1000000000000000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	assert.Equal(t, "1090225464022454202", sim.Info.Reserves[0].String())
	assert.Equal(t, "579100167469821757516857918", sim.Info.Reserves[1].String())

	// the same buy again, on the moved curve, buys less
	second, err := buy(sim, "1000000000000000000")
	require.NoError(t, err)
	assert.Less(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), 0)

	// the clone did not move
	again, err := buy(clone, "1000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, first.TokenAmountOut.Amount, again.TokenAmountOut.Amount)
	assert.Equal(t, "100225464022454202", clone.Info.Reserves[0].String())

	// quoting is pure: the same question gets the same answer
	third, err := buy(sim, "1000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, second.TokenAmountOut.Amount, third.TokenAmountOut.Amount)

	// a round trip lands the curve where it started: fees go to the launchpad's buckets, not to
	// the reserve, and the sell's division rounds up in the seller's favour by at most a wei
	sold, err := sell(sim, first.TokenAmountOut.Amount.String())
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: sold.SwapInfo})
	assert.Equal(t, "937367910787696446494526649", sim.Info.Reserves[1].String())
	drift := new(big.Int).Sub(bigStr("100225464022454202"), sim.Info.Reserves[0])
	assert.True(t, drift.Sign() >= 0 && drift.Cmp(big.NewInt(2)) < 0, "reserve drifted by %s wei", drift)

	// a wrong swap info is ignored rather than applied
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: "nonsense"})
	assert.Equal(t, "937367910787696446494526649", sim.Info.Reserves[1].String())
}

func TestMeta(t *testing.T) {
	sim := newSlyngSimulator(t)

	assert.Equal(t, "", sim.GetApprovalAddress(slyngQuote, slyngToken), "ETH goes as msg.value")
	assert.Equal(t, slyngLaunchpad, sim.GetApprovalAddress(slyngToken, slyngQuote), "a sell is pulled")
	assert.True(t, sim.SwapReceiveNativeIn(slyngQuote, slyngToken, 4663))
	assert.False(t, sim.SwapReceiveNativeIn(slyngToken, slyngQuote, 4663))
	assert.True(t, sim.SwapReturnNativeOut(slyngToken, slyngQuote, 4663))
	assert.False(t, sim.SwapReturnNativeOut(slyngQuote, slyngToken, 4663))

	meta := sim.GetMetaInfo(slyngQuote, slyngToken).(PoolMeta)
	assert.Equal(t, PoolMeta{Launchpad: slyngLaunchpad, Token: slyngToken, ApprovalAddress: "", IsBuy: true,
		IsNativeQuote: true, BlockNumber: 64859506}, meta)
	meta = sim.GetMetaInfo(slyngToken, slyngQuote).(PoolMeta)
	assert.Equal(t, slyngLaunchpad, meta.ApprovalAddress)
	assert.False(t, meta.IsBuy)

	// an ERC-20 quote is approved both ways and moves no native value
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(slyngPoolJSON), &ep))
	ep.Extra = `{"quoteReserve":"250000000","tokenReserve":"900000000000000000000000000","graduated":false}`
	ep.StaticExtra = `{"launchpad":"` + slyngLaunchpad + `","isNativeQuote":false,"graduationTarget":"12545000000","virtualQuote":"3763500000","createdAt":1789435818,"tradeFeeBps":100,"snipeBps":5000,"snipeWindowSeconds":30}`
	erc20, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	assert.Equal(t, slyngLaunchpad, erc20.GetApprovalAddress(slyngQuote, slyngToken))
	assert.False(t, erc20.SwapReceiveNativeIn(slyngQuote, slyngToken, 4663))
	assert.False(t, erc20.SwapReturnNativeOut(slyngToken, slyngQuote, 4663))
	atTime(t, fixtureNow)
	res, err := buy(erc20, "1000000")
	require.NoError(t, err)
	assert.EqualValues(t, defaultGas.BuyERC20, res.Gas)
}
