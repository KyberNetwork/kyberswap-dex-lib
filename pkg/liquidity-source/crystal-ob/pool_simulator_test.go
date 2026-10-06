package crystalob

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Fixtures tracked from a Monad fork at block 111073992. Market 0x39fa (USDC/WMON) is order book
// only and has a bid level at 28000000 holding three orders (50, 30, 7.000001 USDC) to exercise
// per-order rounding. Market 0x2c90 (WMON/graduated launchpad token) has the AMM plus one bid.
// Expected values are eth_call results of Crystal.swap(exactInput, orderType 0, worstPrice 0).
const (
	usdc  = "0x754704bc059f8c67012fed69bc8a327a5aafb603"
	wmon  = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	token = "0x3ebbbaa60c309ec7d857a399051a5e20eb886215"

	bookPoolJSON = `{"address": "0x39fae95717cfd4bda22317f1c124660a166b6bec", "exchange": "crystal-ob", "type": "crystal-ob", "reserves": ["4009006834", "31716538856259954177286"], "tokens": [{"address": "0x754704bc059f8c67012fed69bc8a327a5aafb603", "swappable": true}, {"address": "0x3bd359c1119da7da1d913d1c4d2b7c461115433a", "swappable": true}], "extra": "{\"b\":[[\"28911000\",\"195628067\"],[\"28900000\",\"391256134\"],[\"28878000\",\"547758588\"],[\"28850000\",\"782512269\"],[\"28843000\",\"704261042\"],[\"28668000\",\"1173768404\"],[\"28000000\",\"50000000\",\"30000000\",\"7000001\"],[\"24985000\",\"13013670\"],[\"23785000\",\"11334890\"],[\"22585000\",\"9762440\"],[\"21873000\",\"10994987\"],[\"21386000\",\"8296705\"],[\"21383000\",\"10748677\"],[\"20893000\",\"10502366\"],[\"20404000\",\"10256559\"],[\"20186000\",\"6936865\"],[\"19914000\",\"10010249\"],[\"19424000\",\"9763938\"],[\"18986000\",\"5683352\"],[\"18934000\",\"9517631\"]],\"a\":[[\"29011000\",\"3111638110178273918976\"],[\"29058000\",\"4356293354249584640000\"],[\"29118000\",\"5600948598320894312448\"],[\"29121000\",\"6223276220356548886528\"],[\"29431000\",\"9334914330534823854080\"],[\"29500000\",\"300000000000000000000\"],[\"35989000\",\"244554702457096945904\"],[\"37000000\",\"158303915360525975552\"],[\"38333000\",\"179411104075262754816\"],[\"39667000\",\"200518292789999566848\"],[\"41000000\",\"221625481504736378880\"],[\"42333000\",\"242732670219473158144\"],[\"43667000\",\"263839858934209937408\"],[\"45000000\",\"284947047648946742603\"],[\"45345000\",\"286690925129562245010\"],[\"46323000\",\"149476831091180863488\"],[\"47631000\",\"166666666666666655744\"],[\"48938000\",\"183856502242152480768\"],[\"32000000000\",\"200000000000000000000\"],[\"2100000000000\",\"1000000000000000000\"],[\"2166100000000\",\"873930348654384223\"],[\"2470500000000\",\"1000000000000000000\"],[\"2488100000000\",\"970314151360475866\"],[\"2510500000000\",\"1000000000000000000\"],[\"2530000000000\",\"1000000000000000000\"],[\"2588000000000\",\"1000000000000000000\"]],\"tf\":100000,\"mr\":99995}", "staticExtra": "{\"t\":1,\"s\":\"1000000000000000000000\",\"ts\":\"1\",\"mp\":\"1000000000000000\",\"r\":\"0x508254c838b2e936b0631440c5c6e3ab3a4a98bd\"}", "blockNumber": 111073992}`
	ammPoolJSON  = `{"address": "0x2c906871eeb9be58f00d4127d7b5eb53fec539a1", "exchange": "crystal-ob", "type": "crystal-ob", "reserves": ["116739929569124381627445", "594202274318705746850106683"], "tokens": [{"address": "0x3bd359c1119da7da1d913d1c4d2b7c461115433a", "swappable": true}, {"address": "0x3ebbbaa60c309ec7d857a399051a5e20eb886215", "swappable": true}], "extra": "{\"b\":[[\"74818\",\"100000000000000000000\"]],\"rq\":\"116639929569124381627445\",\"rb\":\"594202274318705746850106683\",\"tf\":99970,\"mr\":99995}", "staticExtra": "{\"t\":4,\"s\":\"1000000000\",\"ts\":\"1\",\"mp\":\"1000000000000000\",\"r\":\"0x508254c838b2e936b0631440c5c6e3ab3a4a98bd\"}", "blockNumber": 111073992}`
)

func newTestSim(t *testing.T, raw string) *PoolSimulator {
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(raw), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// quote returns (amountOut, amountIn actually consumed).
func quote(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut, amountIn string) (*pool.CalcAmountOutResult, string,
	string) {
	amt := bignumber.NewBig(amountIn)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut})
	require.NoError(t, err)
	return res, res.TokenAmountOut.Amount.String(), new(big.Int).Sub(amt, res.RemainingTokenAmountIn.Amount).String()
}

func TestCalcAmountOut_MatchesOnChain(t *testing.T) {
	t.Parallel()
	book, amm := newTestSim(t, bookPoolJSON), newTestSim(t, ammPoolJSON)
	for _, tc := range []struct {
		name              string
		sim               *PoolSimulator
		tokenIn, tokenOut string
		amountIn          string
		wantIn, wantOut   string
	}{
		// taker buys walk the asks; fee is 0 on this market but the maker rebate rounds per order
		{"buy dust", book, usdc, wmon, "1000", "1000", "34435214229085519"},
		{"buy 1 level", book, usdc, wmon, "1000000", "1000000", "34467960428802867877"},
		{"buy several levels", book, usdc, wmon, "1000000000", "1000000000", "31511105071384939317197"},
		{"buy whole book", book, usdc, wmon, "100000000000", "23862267570", "31716538856259954177286"},
		// taker sells walk the bids
		{"sell dust", book, wmon, usdc, "100000000000000", "100000000000000", "2"},
		{"sell 1 level", book, wmon, usdc, "1000000000000000000000", "1000000000000000000000", "28909554"},
		{"sell into 1st of 3 orders", book, wmon, usdc, "132000000000000000000000", "132000000000000000000000",
			"3801806700"},
		{"sell into 2nd of 3 orders", book, wmon, usdc, "134000000000000000000000", "134000000000000000000000",
			"3857803900"},
		{"sell whole book", book, wmon, usdc, "1000000000000000000000000", "140850406860058180047050", "4009006834"},
		// AMM market, 0.03% taker fee charged in quote (WMON)
		{"amm buy dust", amm, wmon, token, "1000", "1000", "5043386"},
		{"amm buy", amm, wmon, token, "1000000000000000000", "1000000000000000000", "5041830750047853778036"},
		{"amm buy large", amm, wmon, token, "1000000000000000000000000", "1000000000000000000000000",
			"531556498245151499866429962"},
		{"amm sell dust", amm, token, wmon, "100000", "100000", "18"},
		{"amm sell", amm, token, wmon, "1000000000000000000000", "1000000000000000000000", "194275082528719536"},
		{"amm sell through bid", amm, token, wmon, "1000000000000000000000000", "1000000000000000000000000",
			"193952262479363781531"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, out, in := quote(t, tc.sim, tc.tokenIn, tc.tokenOut, tc.amountIn)
			assert.Equal(t, tc.wantOut, out)
			assert.Equal(t, tc.wantIn, in)
		})
	}
}

// TestUpdateBalance replays swaps executed in sequence on the fork: the follow-up quotes only
// match if UpdateBalance removed the consumed orders and moved the AMM reserves exactly.
func TestUpdateBalance_MatchesSequentialSwaps(t *testing.T) {
	t.Parallel()
	book := newTestSim(t, bookPoolJSON)
	clone := book.CloneState().(*PoolSimulator)
	res, out, _ := quote(t, book, wmon, usdc, "132000000000000000000000")
	require.Equal(t, "3801806700", out)
	book.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	_, out, _ = quote(t, book, wmon, usdc, "2000000000000000000000")
	assert.Equal(t, "55997200", out)
	_, out, _ = quote(t, book, usdc, wmon, "1000000000")
	assert.Equal(t, "31511105071384939317197", out) // asks untouched
	_, out, _ = quote(t, clone, wmon, usdc, "134000000000000000000000")
	assert.Equal(t, "3857803900", out) // clone keeps the pre-swap book

	amm := newTestSim(t, ammPoolJSON)
	res, out, _ = quote(t, amm, wmon, token, "1000000000000000000000")
	require.Equal(t, "4999452609481390360279105", out)
	amm.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	_, out, _ = quote(t, amm, token, wmon, "1000000000000000000000000")
	assert.Equal(t, "197271620427828399183", out)
	_, out, _ = quote(t, amm, wmon, token, "1000000000000000000000")
	assert.Equal(t, "4915612248887060173473028", out)
}

func TestCalcAmountOut_Errors(t *testing.T) {
	t.Parallel()
	book := newTestSim(t, bookPoolJSON)
	_, err := book.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(1)}, TokenOut: usdc})
	assert.ErrorIs(t, err, ErrInvalidToken)
	_, err = book.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wmon, Amount: big.NewInt(1)}, TokenOut: usdc})
	assert.ErrorIs(t, err, ErrInsufficientBook) // rounds to zero output
	_, err = book.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wmon, Amount: big.NewInt(0)}, TokenOut: usdc})
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = book.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wmon, Amount: new(big.Int).Lsh(big.NewInt(1), 128)}, TokenOut: usdc})
	assert.ErrorIs(t, err, ErrAmountTooLarge) // _marketOrder requires size <= type(uint128).max

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(bookPoolJSON), &ep))
	ep.Extra = "{}" // bonding-curve launchpad market: tracker stores no fees
	_, err = NewPoolSimulator(ep)
	assert.ErrorIs(t, err, ErrMarketInactive)
}

func TestDefaultBuyWorstPrice(t *testing.T) {
	t.Parallel()
	// marketType 1/4 with maxPrice 1e15, tickSize 1: tick 1e6 - 1 -> 99999 * 1e10
	p, ok := defaultBuyWorstPrice(1, big256.U1, big256.TenPow(15))
	require.True(t, ok)
	assert.Equal(t, "999990000000000", p.Dec())
}

// TestMsgpackRoundTrip: router-service loads simulators from msgpack (pkg/msgpack sets
// IncludeUnexported), so every unexported field must survive or quotes silently change.
func TestMsgpackRoundTrip(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{bookPoolJSON, ammPoolJSON} {
		sim := newTestSim(t, raw)
		var buf bytes.Buffer
		enc := msgpack.NewEncoder(&buf)
		enc.IncludeUnexported(true)
		enc.SetForceAsArray(true)
		require.NoError(t, enc.Encode(sim))
		dec := msgpack.NewDecoder(&buf)
		dec.IncludeUnexported(true)
		var decoded PoolSimulator
		require.NoError(t, dec.Decode(&decoded))
		for i := range 2 {
			in, out := sim.Info.Tokens[i], sim.Info.Tokens[1-i]
			_, want, _ := quote(t, sim, in, out, "1000000000000000000000")
			_, got, _ := quote(t, &decoded, in, out, "1000000000000000000000")
			assert.Equal(t, want, got)
		}
	}
}
