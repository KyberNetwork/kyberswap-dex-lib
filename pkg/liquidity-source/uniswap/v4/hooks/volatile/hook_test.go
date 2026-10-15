package volatile

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Real ETH/USDC pool evidence (drpc, blocks 26191342..26195891): per active block b, feeState at the
// previous active block (what the tracker stores), the pool price then, and every swap of b with the fee
// it paid (Swap event) and its post-swap price. "after" is the on-chain feeState at b.
type fixtureState struct {
	SqrtPriceTop string `json:"sqrtPriceTop"`
	AskPerm      uint32 `json:"askPerm"`
	Temp         uint32 `json:"temp"`
	TempOnBid    bool   `json:"tempOnBid"`
}

type fixtureCase struct {
	Block       uint64       `json:"block"`
	State       fixtureState `json:"state"`
	After       fixtureState `json:"after"`
	PriceBefore string       `json:"priceBefore"`
	Swaps       []struct {
		Zfo  bool   `json:"zfo"`
		Post string `json:"post"`
		Fee  uint32 `json:"fee"`
	} `json:"swaps"`
}

type fixtureFile struct {
	Config map[string]uint32 `json:"config"`
	Cases  []fixtureCase     `json:"cases"`
}

// liveConfig is feeConfig(poolId) on mainnet (unchanged over the evidence blocks).
var liveConfig = FeeConfig{RestingSpread: 800, BackrunCaptureFraction: 500000, MidFadeRate: 4000000, MaxMidFade: 200,
	SkewRate: 1000000, MaxSkewStep: 300, MaxSkew: 800, BackrunThreshold: 500, RestoreFraction: 1000000, RestoreThreshold: 500}

const benchHookExtra = `{"c":{"rs":800,"bc":500000,"mr":4000000,"mm":200,"sr":1000000,"ss":300,"ms":800,"bt":500,"rf":1000000,"rt":500},` +
	`"s":{"p":"3845578974581658764126876","a":300}}`

func (s fixtureState) feeState(t testing.TB) FeeState {
	var fs FeeState
	require.NoError(t, fs.SqrtPriceTop.SetFromDecimal(s.SqrtPriceTop))
	fs.AskPermFee, fs.TempFee, fs.TempOnBid = s.AskPerm, s.Temp, s.TempOnBid
	return fs
}

func loadFixture(t testing.TB) []fixtureCase {
	raw, err := os.ReadFile("testdata/fixture.json")
	require.NoError(t, err)
	var f fixtureFile
	require.NoError(t, json.Unmarshal(raw, &f))
	require.Equal(t, liveConfig.RestingSpread, f.Config["restingSpread"])
	require.Equal(t, liveConfig.MaxMidFade, f.Config["maxMidFade"])
	return f.Cases
}

func newHook(state FeeState) *Hook {
	return &Hook{Hook: &uniswapv4.BaseHook{Exchange: DexType}, Extra: Extra{Config: liveConfig, State: state}}
}

func beforeSwap(t testing.TB, h *Hook, price *uint256.Int, zeroForOne bool) *uniswapv4.BeforeSwapResult {
	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{ZeroForOne: zeroForOne, CalcOut: true,
		AmountSpecified: big.NewInt(1), SqrtPriceX96: price})
	require.NoError(t, err)
	return res
}

// The tracker stores feeState at the previous active block P. The next active block's first swap pays the fee
// computed from the advance of that state, not getFee@P (which differs in 97 of 119 sampled pairs): the sim
// must advance the stored state and reproduce the paid fee exactly. Later legs must pay the fee the chain
// charged them, from the advanced state and the post-swap price of the previous leg.
func TestBeforeSwap_ReproducesOnchainFees(t *testing.T) {
	var first, later int
	for _, c := range loadFixture(t) {
		h := newHook(c.State.feeState(t))
		price := uint256.MustFromDecimal(c.PriceBefore)
		for i, sw := range c.Swaps {
			res := beforeSwap(t, h, price, sw.Zfo)
			require.Equal(t, sw.Fee, uint32(res.SwapFee), "block %d swap %d", c.Block, i)
			if i == 0 {
				first++
				// the advance equals the state the hook persisted on chain at this block
				info, ok := res.SwapInfo.(*SwapInfo)
				require.True(t, ok, "block %d", c.Block)
				require.Equal(t, c.After.feeState(t), info.State, "block %d advanced state", c.Block)
			} else {
				later++
				require.Nil(t, res.SwapInfo, "block %d: no second advance in a block", c.Block)
			}
			h.UpdateBalance(res.SwapInfo)
			price = uint256.MustFromDecimal(sw.Post)
		}
	}
	t.Logf("first-in-block swaps matched: %d, later legs matched: %d", first, later)
}

// Without the in-block state a second same-direction leg is under-priced by up to maxMidFade (+200 here).
func TestBeforeSwap_SecondLegPaysMore(t *testing.T) {
	var multi int
	for _, c := range loadFixture(t) {
		if len(c.Swaps) < 2 || c.Swaps[0].Zfo != c.Swaps[1].Zfo {
			continue
		}
		multi++
		zfo := c.Swaps[0].Zfo
		h := newHook(c.State.feeState(t))
		leg1 := beforeSwap(t, h, uint256.MustFromDecimal(c.PriceBefore), zfo)
		h.UpdateBalance(leg1.SwapInfo)
		post := uint256.MustFromDecimal(c.Swaps[0].Post)

		leg2 := beforeSwap(t, h, post, zfo)
		require.Greater(t, leg2.SwapFee, leg1.SwapFee, "block %d", c.Block)
		require.LessOrEqual(t, uint32(leg2.SwapFee-leg1.SwapFee), liveConfig.MaxMidFade, "block %d", c.Block)

		// a hook that never saw leg 1 (flat fee model) would advance again at the post-swap price
		stale := newHook(c.State.feeState(t))
		require.NotEqual(t, c.Swaps[1].Fee, uint32(beforeSwap(t, stale, post, zfo).SwapFee), "block %d", c.Block)
	}
	require.NotZero(t, multi)
}

// CloneState must isolate route branches: UpdateBalance on a clone leaves the original quoting as a first leg.
func TestCloneState_IsolatesAdvance(t *testing.T) {
	c := loadFixture(t)[0]
	h := newHook(c.State.feeState(t))
	price := uint256.MustFromDecimal(c.PriceBefore)
	clone := h.CloneState().(*Hook)
	res := beforeSwap(t, clone, price, c.Swaps[0].Zfo)
	clone.UpdateBalance(res.SwapInfo)
	require.True(t, clone.advanced)

	require.False(t, h.advanced)
	require.Equal(t, c.State.feeState(t), h.State)
	require.NotNil(t, beforeSwap(t, h, price, c.Swaps[0].Zfo).SwapInfo)
}

// Displacement within a block (hand-computed with exact integer math from VolatileFeeCalculation.sol):
// top 3.9e24, price moved -0.5% / +0.5% in sqrt terms, askPerm 300 (bid 500), no temp fee.
// Backrun side pays baseline + ceil(capture * move); the continuing side pays baseline + maxMidFade.
func TestBeforeSwap_DisplacementBackrunAndContinuing(t *testing.T) {
	state := FeeState{AskPermFee: 300}
	state.SqrtPriceTop.SetFromDecimal("3900000000000000000000000")
	for _, tc := range []struct {
		name       string
		price      string
		sell, buy  uint32
		sellIsBack bool
	}{
		{"price down", "3880500000000000000000000", 700, 5288, false},
		{"price up", "3919500000000000000000000", 5513, 500, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHook(state)
			h.advanced = true // in-block state: no new advance
			price := uint256.MustFromDecimal(tc.price)
			require.Equal(t, tc.sell, uint32(beforeSwap(t, h, price, true).SwapFee))
			require.Equal(t, tc.buy, uint32(beforeSwap(t, h, price, false).SwapFee))
		})
	}
}

// A config reset leaves sqrtPriceTop = 0: the first swap only records the block-open price.
func TestBeforeSwap_AfterConfigReset(t *testing.T) {
	h := newHook(FeeState{AskPermFee: liveConfig.RestingSpread / 2})
	price := uint256.MustFromDecimal("3900000000000000000000000")
	res := beforeSwap(t, h, price, true)
	require.EqualValues(t, 400, res.SwapFee)
	require.Equal(t, *price, res.SwapInfo.(*SwapInfo).State.SqrtPriceTop)
}

func TestBeforeSwap_Errors(t *testing.T) {
	_, err := (&Hook{}).BeforeSwap(&uniswapv4.BeforeSwapParams{SqrtPriceX96: uint256.NewInt(1)})
	require.ErrorIs(t, err, ErrPoolIsNotTracked)

	_, err = newHook(FeeState{}).BeforeSwap(&uniswapv4.BeforeSwapParams{})
	require.ErrorIs(t, err, ErrInvalidSqrtPrice)
}

// Prod holds hX in the old flat shape ({"t","f0","f1"}) until the tracker rewrites it: that must not quote.
func TestHookFactory_LegacyExtraIsUntracked(t *testing.T) {
	hook, ok := uniswapv4.GetHook(HookAddress, &uniswapv4.HookParam{
		HookExtra: uniswapv4.HookExtra(`{"t":true,"f0":800,"f1":263}`)})
	require.True(t, ok)
	_, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{SqrtPriceX96: uint256.NewInt(1)})
	require.ErrorIs(t, err, ErrPoolIsNotTracked)
}

func TestHookExtra_JSONRoundTrip(t *testing.T) {
	c := loadFixture(t)[0]
	want := newHook(c.State.feeState(t))
	raw, err := json.Marshal(want)
	require.NoError(t, err)

	hook, _ := uniswapv4.GetHook(HookAddress, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)})
	require.Equal(t, want.Extra, hook.(*Hook).Extra)
}

// feeState/feeConfig return several uintN outputs: ethrpc unpacks them only into one abi-tagged struct
// (two pointers failed with "cannot unmarshal tuple" against the live hook).
func TestStateABI_UnpacksIntoTaggedStruct(t *testing.T) {
	data, err := volatilePairHookABI.Methods["feeState"].Outputs.Pack(
		big.NewInt(26195891), big.NewInt(12345), big.NewInt(300), big.NewInt(7), true)
	require.NoError(t, err)
	var state struct {
		BlockNumber  *big.Int `abi:"blockNumber"` // unused: a sim swap always runs in a later block
		SqrtPriceTop *big.Int `abi:"sqrtPriceTopX96"`
		AskPermFee   *big.Int `abi:"askPermFeeE6"`
		TempFee      *big.Int `abi:"tempFeeE6"`
		TempOnBid    bool     `abi:"tempOnBid"`
	}
	require.NoError(t, volatilePairHookABI.UnpackIntoInterface(&state, "feeState", data))
	require.EqualValues(t, 12345, state.SqrtPriceTop.Int64())
	require.EqualValues(t, 300, state.AskPermFee.Int64())
	require.EqualValues(t, 7, state.TempFee.Int64())
	require.True(t, state.TempOnBid)
}

// Through the real pool simulator: the price reaching the hook is the pool's (moved by leg 1), the first
// leg advances the state once, the second leg does not, and a clone is isolated.
func TestPoolSimulator_TwoLegsAdvanceOnce(t *testing.T) {
	sim := newBenchSim(t, benchHookExtra)
	const weth, usdc = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(5e18)}, TokenOut: usdc}

	leg1, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotNil(t, leg1.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo)

	clone := sim.CloneState().(*uniswapv4.PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *leg1.TokenAmountOut,
		Fee: *leg1.Fee, SwapInfo: leg1.SwapInfo})

	leg2, err := clone.CalcAmountOut(params)
	require.NoError(t, err)
	require.Nil(t, leg2.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo)

	again, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.NotNil(t, again.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo)
	require.Equal(t, leg1.TokenAmountOut.Amount, again.TokenAmountOut.Amount)
}
