package hyfiprop

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/hooktest"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// hooktest's pool: token0 (0x109d...) is the pair's base here, token1 the quote.
const tokenBase, tokenQuote = "0x109ddc73b46b2f8141880a5573d2b0d2acf10b07", "0x4200000000000000000000000000000000000006"

type seqFixture struct {
	TickWidth    string    `json:"tickWidth"`
	BaseLiqUnit  string    `json:"baseLiqUnit"`
	FeePerSecond uint32    `json:"feePerSecond"`
	BaseIsC0     bool      `json:"baseIsCurrency0"`
	Bid          [3]string `json:"bid"`
	Ask          [3]string `json:"ask"`
	Steps        []struct {
		Now        uint64 `json:"now"`
		ZeroForOne bool   `json:"zeroForOne"`
		In         string `json:"in"`
		OK         bool   `json:"ok"`
		Out        string `json:"out"`
		Fee        string `json:"fee"`
	} `json:"steps"`
}

// ten swapExactInDirect calls in a row on an anvil fork of Base against the real HyFi contract, with a
// WETH/USDC pair and a 68-tick book on each side (see testdata/swap_sequence.json).
func loadSeq(t testing.TB) (*seqFixture, *Hook) {
	t.Helper()
	raw, err := os.ReadFile("testdata/swap_sequence.json")
	require.NoError(t, err)
	var f seqFixture
	require.NoError(t, json.Unmarshal(raw, &f))
	h := &Hook{
		Cfg:       PairConfig{TickWidth: u(f.TickWidth), BaseLiqUnit: u(f.BaseLiqUnit), FeePerSecond: f.FeePerSecond, BaseIsC0: f.BaseIsC0},
		Book:      [2]Side{hexSide(t, f.Bid), hexSide(t, f.Ask)},
		Inventory: [2]uint256.Int{u("1000000000000000000000000"), u("1000000000000000000000000")},
	}
	return &f, h
}

func setNow(t testing.TB, now uint64) {
	t.Helper()
	unixNow = func() uint64 { return now }
	t.Cleanup(func() { unixNow = defaultUnixNow })
}

// The walk pointer a trade leaves in slot0 decides the next trade's price. Only UpdateBalance carries it,
// so a wrong pointer shows up as a wrong amount in a later step.
func TestUpdateBalance_WalkPointerMatchesChain(t *testing.T) {
	f, h := loadSeq(t)
	for i, s := range f.Steps {
		setNow(t, s.Now)
		amt := u(s.In)
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: s.ZeroForOne,
			AmountSpecified: amt.ToBig()})
		if s.Out == "0" { // the chain still walks; the simulator refuses zero-output swaps
			require.ErrorIs(t, err, ErrZeroOutput)
			r, err := priceExactIn(&h.Book, &h.Cfg, s.ZeroForOne, &amt, s.Now)
			require.NoError(t, err)
			h.UpdateBalance(&SwapInfo{ZeroForOne: s.ZeroForOne, IsSellingBase: r.isSellingBase, Cur: r.cur, Left: r.left,
				AmountIn: amt, AmountOut: r.amountOut})
			continue
		}
		require.NoError(t, err, "step %d", i)
		assert.Equal(t, s.Out, new(big.Int).Neg(res.DeltaUnspecified).String(), "step %d out", i)
		h.UpdateBalance(res.SwapInfo)
	}
}

// A swap quoted on a clone must not move the original's pointer or inventory.
func TestCloneState_Independent(t *testing.T) {
	f, h := loadSeq(t)
	setNow(t, f.Steps[0].Now)
	before := *h
	clone := h.CloneState().(*Hook)
	amt := u(f.Steps[0].In)
	res, err := clone.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: amt.ToBig()})
	require.NoError(t, err)
	clone.UpdateBalance(res.SwapInfo)
	assert.Equal(t, before.Book, h.Book)
	assert.Equal(t, before.Inventory, h.Inventory)
	assert.NotEqual(t, before.Book, clone.Book)
	assert.NotEqual(t, before.Inventory, clone.Inventory)
}

func TestBeforeSwap_Refusals(t *testing.T) {
	f, h := loadSeq(t)
	setNow(t, f.Steps[0].Now)
	swap := func(zfo bool, amt string) error {
		a := u(amt)
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo, AmountSpecified: a.ToBig()})
		return err
	}
	assert.ErrorIs(t, swap(true, "0"), ErrInvalidAmount)
	assert.ErrorIs(t, swap(true, "1"), ErrZeroOutput)
	assert.ErrorIs(t, swap(true, "1000000000000000000000000"), ErrInsufficientLiquidity)

	// the quote token the contract pays out is capped by its balance
	h.Inventory[1] = u("1000000")
	assert.ErrorIs(t, swap(true, "1000000000000000000"), ErrInsufficientInv)
	assert.NoError(t, swap(true, "100000000000000"))

	// a book older than 1e6/feePerSecond seconds is all fee
	setNow(t, f.Steps[0].Now+1_000_000/uint64(f.FeePerSecond))
	h.Inventory[1] = u("1000000000000000000000000")
	assert.ErrorIs(t, swap(true, "1000000000000000000"), ErrZeroOutput)

	_, err := (&Hook{}).BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(1e9)})
	assert.ErrorIs(t, err, ErrPairNotConfigured, "an untracked pair has no tick width")
}

// CalcAmountIn inverts the exact-in quote: the min input whose output reaches the target.
func TestCalcAmountIn_RoundTrip(t *testing.T) {
	f, h := loadSeq(t)
	setNow(t, f.Steps[0].Now)
	sim := hooktest.NewPoolSimulator(t, HookAddresses[0], h)
	for _, out := range []string{"1000000", "123456789", "2000000000"} {
		want, _ := new(big.Int).SetString(out, 10)
		in, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenIn: tokenBase,
			TokenAmountOut: pool.TokenAmount{Token: tokenQuote, Amount: want}})
		require.NoError(t, err)
		got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenOut: tokenQuote,
			TokenAmountIn: pool.TokenAmount{Token: tokenBase, Amount: in.TokenAmountIn.Amount}})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, got.TokenAmountOut.Amount.Cmp(want), 0)
		less := new(big.Int).Sub(in.TokenAmountIn.Amount, big.NewInt(1))
		if got, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenOut: tokenQuote,
			TokenAmountIn: pool.TokenAmount{Token: tokenBase, Amount: less}}); err == nil {
			assert.Negative(t, got.TokenAmountOut.Amount.Cmp(want), "input is not minimal")
		}
	}
}

// Through the full v4 simulator: the hook's quote is the pool's quote, gas is the direct path's, and the
// hook state survives the JSON round trip pool-service stores.
func TestPoolSimulator_QuotesAndGas(t *testing.T) {
	f, h := loadSeq(t)
	setNow(t, f.Steps[0].Now)
	sim := hooktest.NewPoolSimulator(t, HookAddresses[0], h)
	s := f.Steps[0]
	in := u(s.In)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenOut: tokenQuote,
		TokenAmountIn: pool.TokenAmount{Token: tokenBase, Amount: in.ToBig()}})
	require.NoError(t, err)
	assert.Equal(t, s.Out, res.TokenAmountOut.Amount.String())
	assert.Equal(t, gasDirect, res.Gas)
	assert.True(t, res.RemainingTokenAmountIn == nil || res.RemainingTokenAmountIn.Amount.Sign() == 0)

	got, err := (&Hook{}).GetReserves(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"0", "0"}, []string(got))

	raw, err := json.Marshal(h)
	require.NoError(t, err)
	var back Hook
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, h.Book, back.Book)
	assert.Equal(t, h.Cfg, back.Cfg)
	assert.Equal(t, h.Inventory, back.Inventory)
}

// hyfiprop's hX holds only chain state, so it must opt in to the tracker's activity bump; without
// it a task-only refresh never moves Timestamp and the pool is archived while the book is live.
var _ uniswapv4.HookTracksActivity = (*Hook)(nil)
