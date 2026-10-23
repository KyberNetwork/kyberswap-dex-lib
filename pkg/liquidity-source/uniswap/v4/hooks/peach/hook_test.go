package peach

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func testHook() *Hook {
	return &Hook{BaseHook: uniswapv4.BaseHook{Exchange: DexType}, ValidDeployment: true, QuoteAt: 1000,
		Extra: Extra{Tracked: true, Timestamp: 1000, Floor: 100, Cap: 6000, AbsoluteCap: 1_000_000,
			Autonomous0For1: 700, Autonomous1For0: 900, Fee0For1: 700, Fee1For0: 900,
			Mid: MidBlockConfig{true, 100, 500_000, 5000}}}
}

func quote(t testing.TB, h *Hook, price string, zfo bool) *uniswapv4.BeforeSwapResult {
	t.Helper()
	r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zfo, AmountSpecified: big.NewInt(100), SqrtPriceX96: uint256.MustFromDecimal(price)})
	require.NoError(t, err)
	return r
}

func TestMidBlockOracleVectors(t *testing.T) {
	b, err := os.ReadFile("testdata/mid_block_vectors.json")
	require.NoError(t, err)
	var fixture struct {
		Cases []struct {
			Open, Current                      string
			ZeroForOne                         bool
			Threshold, Fraction, Cap, Expected uint32
		}
	}
	require.NoError(t, json.Unmarshal(b, &fixture))
	require.Len(t, fixture.Cases, 384)
	for i, c := range fixture.Cases {
		open, current := uint256.MustFromDecimal(c.Open), uint256.MustFromDecimal(c.Current)
		got := surcharge(open, current, c.ZeroForOne, MidBlockConfig{true, c.Threshold, c.Fraction, c.Cap})
		require.Equal(t, c.Expected, got, "oracle case %d", i)
	}
}

func TestMidBlockSequenceAndQuotePurity(t *testing.T) {
	h := testHook()
	first := quote(t, h, "1000000", true)
	require.EqualValues(t, 700, first.SwapFee)
	require.False(t, h.Advanced)
	require.Equal(t, *first, *quote(t, h, "1000000", true), "repeat quote must not seed state")
	h.UpdateBalance(first.SwapInfo)
	require.True(t, h.Advanced)
	// sqrt +1% => price +2.01%; floor(20100*0.5)=10050, bounded by midCap=5000.
	require.EqualValues(t, 5700, quote(t, h, "1010000", true).SwapFee)
	require.EqualValues(t, 900, quote(t, h, "1010000", false).SwapFee, "outward swaps pay no surcharge")
	require.EqualValues(t, 5900, quote(t, h, "990000", false).SwapFee)
	require.EqualValues(t, 700, quote(t, h, "990000", true).SwapFee)
	require.EqualValues(t, 700, quote(t, h, "1000000", true).SwapFee)
	// A failed/unaccepted quote never changes the opening reference.
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{})
	require.ErrorIs(t, err, ErrInvalidPrice)
	require.Equal(t, uint256.NewInt(1000000), &h.OpenPrice)
}

func TestCloneIsolatesStateAndDisabledMidBlock(t *testing.T) {
	h := testHook()
	clone := h.CloneState().(*Hook)
	clone.UpdateBalance(quote(t, clone, "1000000", true).SwapInfo)
	require.False(t, h.Advanced)
	require.True(t, clone.Advanced)
	clone.Cap = 1000
	require.EqualValues(t, 1000, quote(t, clone, "1010000", true).SwapFee)
	require.EqualValues(t, 6000, h.Cap)
	h.Mid.Enabled = false
	r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: true})
	require.NoError(t, err)
	require.EqualValues(t, 700, r.SwapFee)
	require.Nil(t, r.SwapInfo)
	after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{})
	require.NoError(t, err)
	require.Zero(t, after.HookFee.Sign())
}

func TestKeeperExpiryPauseAndScheduling(t *testing.T) {
	h := testHook()
	h.Poke0For1 = 200
	h.Poke1For0 = 2000
	h.PokeExpiry = 1001
	require.EqualValues(t, 350, quote(t, h, "1000", true).SwapFee, "live override cannot discount over 50%")
	require.EqualValues(t, 2000, quote(t, h, "1000", false).SwapFee)
	require.EqualValues(t, 1001, h.NextTrackAt(nil))
	h.QuoteAt = 1001
	require.EqualValues(t, 700, quote(t, h, "1000", true).SwapFee, "expiry is exclusive")
	h.PausedUntil = 1010
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{})
	require.ErrorIs(t, err, ErrPaused)
	require.EqualValues(t, 1010, h.NextTrackAt(nil))
	h.QuoteAt = 1010
	h.PoolPausedUntil = 1020
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{})
	require.ErrorIs(t, err, ErrPaused)
	h.QuoteAt = 1020
	quote(t, h, "1000", true)
	h.RWA = true
	require.EqualValues(t, 1021, h.NextTrackAt(nil))
}

func TestRegistrationChainScopeAndJSON(t *testing.T) {
	for address, d := range deployments {
		extra, err := json.Marshal(testHook().Extra)
		require.NoError(t, err)
		params := &uniswapv4.HookParam{HookAddress: address, Cfg: &uniswapv4.Config{ChainID: d.ChainID}, HookExtra: extra, QuoteAt: 1000}
		h, ok := uniswapv4.GetHook(address, params)
		require.True(t, ok)
		require.Equal(t, string(DexType), h.GetExchange())
		quote(t, h.(*Hook), "1000", true)
		params.Cfg.ChainID = 1
		wrong, _ := uniswapv4.GetHook(address, params)
		_, err = wrong.BeforeSwap(&uniswapv4.BeforeSwapParams{})
		require.ErrorIs(t, err, ErrDeployment)
	}
	_, ok := uniswapv4.GetHook(common.HexToAddress("0x1234"), &uniswapv4.HookParam{})
	require.False(t, ok)
}

func TestOrdinaryFeeBounds(t *testing.T) {
	for _, c := range []struct {
		auto, floor, cap, poke uint32
		active                 bool
		want                   uint32
	}{
		{700, 100, 6000, 0, false, 700}, {700, 100, 6000, 100, true, 350},
		{101, 100, 6000, 1, true, 100}, {700, 100, 6000, 9000, true, 6000},
		{9000, 100, 6000, 0, false, 6000}, {9000, 100, 6000, 100, true, 3000},
	} {
		require.Equal(t, c.want, resolveFee(c.auto, c.floor, c.cap, c.poke, c.active))
	}
}

func TestMalformedAndForeignExtraRefuseQuotes(t *testing.T) {
	address := common.HexToAddress("0x4D1546682BC8252E52c1c1820F0AaA7694C9e080")
	for _, raw := range []string{`{"ok":true,"cap":"bad"}`, `{"f01":700,"f10":700}`, `{"ok":true,`, ``} {
		h, _ := uniswapv4.GetHook(address, &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: 56}, HookExtra: uniswapv4.HookExtra(raw)})
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{})
		require.ErrorIs(t, err, ErrUntracked)
	}
}

func BenchmarkSurcharge(b *testing.B) {
	open := uint256.MustFromDecimal("2275579071148052468332900315181")
	current := uint256.MustFromDecimal("2285579071148052468332900315181")
	config := MidBlockConfig{true, 1, 500_000, 1_000_000}
	b.ReportAllocs()
	for b.Loop() {
		_ = surcharge(open, current, true, config)
	}
}
