package kalos

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func fixturePools(t *testing.T) []fixturePool {
	t.Helper()
	raw, err := os.ReadFile("testdata/pools.json")
	require.NoError(t, err)
	var pools []fixturePool
	require.NoError(t, json.Unmarshal(raw, &pools))
	require.Len(t, pools, 2)
	return pools
}

// unhooked is the same pool without a hook, at a static LP fee: what the PoolManager's own swap gives.
func unhooked(t *testing.T, p entity.Pool, fee uint32) *uniswapv4.PoolSimulator {
	t.Helper()
	var se map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	se["hooks"], _ = json.Marshal(common.Address{})
	se["fee"], _ = json.Marshal(fee)
	b, err := json.Marshal(se)
	require.NoError(t, err)
	p.StaticExtra, p.SwapFee, p.Exchange = string(b), float64(fee), uniswapv4.DexType
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDElysium)
	require.NoError(t, err)
	return sim
}

func out(t *testing.T, sim pool.IPoolSimulator, p entity.Pool, zeroForOne bool, amountIn *big.Int) *big.Int {
	t.Helper()
	amount, err := outErr(sim, p, zeroForOne, amountIn)
	require.NoError(t, err)
	return amount
}

func outErr(sim pool.IPoolSimulator, p entity.Pool, zeroForOne bool, amountIn *big.Int) (*big.Int, error) {
	tokenIn, tokenOut := tokensOf(p, zeroForOne)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

// Every kind of pool the hook prices, on the live pools' liquidity: the quote is the PoolManager's swap at the LP fee
// splitFee gives, with the hook's part ceil(h * gross) of the fee asset taken from the input (beforeSwap) or the
// output (afterSwap); a plain pool is its calm minimum only.
func TestKinds(t *testing.T) {
	t.Parallel()
	creator := FeeParams{MinFeeZeroForOne: 10_000, MaxFeeZeroForOne: 10_000, MinFeeOneForZero: 10_000,
		MaxFeeOneForZero: 10_000, FilterPeriod: 1, DecayPeriod: 2, ResetPeriod: 1}
	for _, fp := range fixturePools(t) {
		tracked, tick := hookExtraOf(t, fp.Pool)
		now := uint64(fp.Pool.Timestamp)
		for name, hx := range map[string]Extra{
			"tracked":         tracked,
			"creator side 0":  {Registered: true, Kind: kindCreator, FeeSide: feeCurrency0, KalosShareBps: 4000, Params: creator},
			"creator side 1":  {Registered: true, Kind: kindCreator, FeeSide: feeCurrency1, KalosShareBps: 4000, Params: creator},
			"no fee asset":    {Registered: true, FeeSide: feeNone, Params: tracked.Params, State: tracked.State},
			"features off":    {Registered: true, FeeSide: tracked.FeeSide, KalosShareBps: 1500, FeaturesOff: true, Params: tracked.Params, State: tracked.State},
			"book frozen":     {Registered: true, FeeSide: tracked.FeeSide, KalosShareBps: 1500, BookFrozen: true, Params: tracked.Params, State: tracked.State},
			"creator, frozen": {Registered: true, Kind: kindCreator, FeeSide: feeCurrency1, KalosShareBps: 4000, BookFrozen: true, FeaturesOff: true, Params: creator},
		} {
			sim, err := uniswapv4.NewPoolSimulator(withHookExtra(t, fp.Pool, hx), valueobject.ChainIDElysium)
			require.NoError(t, err)
			for _, zfo := range []bool{true, false} {
				var h, lp uint32
				if hx.plain() {
					lp = hx.Params.calmFee(zfo)
				} else {
					total, _, _, ok := hx.Params.computeFee(hx.State, tick, now, zfo)
					require.True(t, ok)
					h, lp = splitFee(hx.Kind, hx.FeeSide, hx.KalosShareBps, total, zfo)
				}
				for _, q := range fp.Quotes {
					if q.ZeroForOne != zfo {
						continue
					}
					amountIn, _ := new(big.Int).SetString(q.AmountIn, 10)
					var want *big.Int
					switch {
					case h == 0:
						want = out(t, unhooked(t, fp.Pool, lp), fp.Pool, zfo, amountIn)
					case hx.feeAssetIn(zfo):
						fee := hookFeeOf(uint256.MustFromBig(amountIn), h)
						want = out(t, unhooked(t, fp.Pool, lp), fp.Pool, zfo, new(big.Int).Sub(amountIn, fee.ToBig()))
					default:
						require.Zero(t, lp)
						gross := out(t, unhooked(t, fp.Pool, 0), fp.Pool, zfo, amountIn)
						fee := hookFeeOf(uint256.MustFromBig(gross), h)
						want = gross.Sub(gross, fee.ToBig())
					}
					got, err := outErr(sim, fp.Pool, zfo, amountIn)
					require.NoError(t, err, "%s %s zfo=%v in=%s want=%s h=%d lp=%d", fp.Name, name, zfo, amountIn, want, h, lp)
					require.Equal(t, want.String(), got.String(), "%s %s zfo=%v in=%s", fp.Name, name, zfo, amountIn)
				}
			}
		}
	}
}

// A quote reverses: CalcAmountIn's input delivers the output, every kind and direction.
func TestKinds_CalcAmountIn(t *testing.T) {
	t.Parallel()
	creator := FeeParams{MinFeeZeroForOne: 10_000, MaxFeeZeroForOne: 10_000, MinFeeOneForZero: 10_000,
		MaxFeeOneForZero: 10_000, FilterPeriod: 1, DecayPeriod: 2, ResetPeriod: 1}
	for _, fp := range fixturePools(t) {
		tracked, _ := hookExtraOf(t, fp.Pool)
		for _, hx := range []Extra{tracked,
			{Registered: true, Kind: kindCreator, FeeSide: feeCurrency0, KalosShareBps: 4000, Params: creator},
			{Registered: true, Kind: kindCreator, FeeSide: feeCurrency1, KalosShareBps: 4000, Params: creator},
			{Registered: true, FeeSide: feeNone, Params: tracked.Params, State: tracked.State},
		} {
			sim, err := uniswapv4.NewPoolSimulator(withHookExtra(t, fp.Pool, hx), valueobject.ChainIDElysium)
			require.NoError(t, err)
			for _, q := range fp.Quotes[1:] {
				tokenIn, tokenOut := tokensOf(fp.Pool, q.ZeroForOne)
				amountIn, _ := new(big.Int).SetString(q.AmountIn, 10)
				target := out(t, sim, fp.Pool, q.ZeroForOne, amountIn)
				resIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: target}, TokenIn: tokenIn})
				require.NoError(t, err)
				back := out(t, sim, fp.Pool, q.ZeroForOne, resIn.TokenAmountIn.Amount)
				require.GreaterOrEqual(t, back.Cmp(target), 0, "%s kind=%d side=%d", fp.Name, hx.Kind, hx.FeeSide)
				// It delivers no more than a millionth over (a wei for a coarse output): an exact-out fee rule in place of
				// the exact-in fee's inverse would be off by about fee^2.
				over := new(big.Int).Sub(back, target)
				require.LessOrEqual(t, over.Mul(over, big.NewInt(1_000_000)).Cmp(new(big.Int).Add(target,
					big.NewInt(1_000_000))), 0, "%s kind=%d side=%d out=%s back=%s", fp.Name, hx.Kind, hx.FeeSide, target, back)
			}
		}
	}
}

func TestRefusals(t *testing.T) {
	t.Parallel()
	fp := fixturePools(t)[0]
	tokenIn, tokenOut := tokensOf(fp.Pool, true)

	sim, err := uniswapv4.NewPoolSimulator(withHookExtra(t, fp.Pool, Extra{}), valueobject.ChainIDElysium)
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(1e15)}, TokenOut: tokenOut})
	require.ErrorIs(t, err, ErrPoolNotRegistered)

	sim, err = uniswapv4.NewPoolSimulator(fp.Pool, valueobject.ChainIDElysium)
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Lsh(big.NewInt(1), 127)}, TokenOut: tokenOut})
	require.ErrorIs(t, err, ErrAmountTooLarge)

	tracked, _ := hookExtraOf(t, fp.Pool)
	tracked.State.ResetTime = uint64(fp.Pool.Timestamp) + 10 // after the swap: the contract's subtraction underflows
	sim, err = uniswapv4.NewPoolSimulator(withHookExtra(t, fp.Pool, tracked), valueobject.ChainIDElysium)
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(1e15)}, TokenOut: tokenOut})
	require.ErrorIs(t, err, ErrInvalidFeeState)
}

// The wall clock is read a second early, and the indexing clock as is.
func TestClock(t *testing.T) {
	orig := NowFn
	NowFn = func() int64 { return 1_794_947_405 }
	t.Cleanup(func() { NowFn = orig })
	require.Equal(t, uint64(1_794_947_404), (&Hook{}).now())
	require.Equal(t, uint64(1_794_947_000), (&Hook{quoteAt: 1_794_947_000}).now())
}

// trackStub is a chain at block head serving the hook's reads for pool, with the PoolManager's slot0 word at
// slot0 (the tracked price and tick when nil).
func trackStub(t *testing.T, head uint64, p entity.Pool, config poolConfigRaw, slot0 *common.Hash) *testutil.MulticallStub {
	t.Helper()
	hx, _ := hookExtraOf(t, p)
	params, state := &hx.Params, &hx.State
	word := slot0Word(t, p)
	if slot0 != nil {
		word = *slot0
	}
	r := testutil.NewMethodResponder(t, kalosDynamicABI, poolManagerABI).
		Set("poolConfig", config.Registered, config.Kind, config.FeeSide, config.Preset, config.KalosShareBps,
			config.LaunchToken, config.BookFrozen).
		Set("feeParams", feeParamsRaw{
			MinFeeZeroForOne: big.NewInt(int64(params.MinFeeZeroForOne)), MaxFeeZeroForOne: big.NewInt(int64(params.MaxFeeZeroForOne)),
			MinFeeOneForZero: big.NewInt(int64(params.MinFeeOneForZero)), MaxFeeOneForZero: big.NewInt(int64(params.MaxFeeOneForZero)),
			Dynamic: params.Dynamic, Directional: params.Directional, FilterPeriod: uint16(params.FilterPeriod),
			DecayPeriod: uint16(params.DecayPeriod), ReductionFactor: uint16(params.ReductionFactor),
			VariableFeeControl:       new(big.Int).SetUint64(params.VariableFeeControl),
			MaxVolatilityAccumulator: new(big.Int).SetUint64(params.MaxVolatilityAccumulator),
			ResetPeriod:              uint16(params.ResetPeriod), ResetTickFilter: new(big.Int).SetUint64(params.ResetTickFilter),
			SameMomentSeconds: uint8(params.SameMomentSeconds),
		}).
		Set("feeState", feeStateRaw{
			ReferenceTick: big.NewInt(int64(state.ReferenceTick)), LastTick: big.NewInt(int64(state.LastTick)),
			Peak: big.NewInt(int64(state.Peak)), VolatilityReference: new(big.Int).SetUint64(state.VolatilityReference),
			LastSwap: new(big.Int).SetUint64(state.LastSwap), ResetTime: new(big.Int).SetUint64(state.ResetTime),
			MoveTime: new(big.Int).SetUint64(state.MoveTime), PeakTime: new(big.Int).SetUint64(state.PeakTime),
		}).
		Set("featuresOff", false).
		Set("extsload", word)
	return testutil.NewMulticallStub(t, r.Respond).At(head, uint64(p.Timestamp))
}

// slot0Word packs the pool's tracked price and tick as the PoolManager stores slot0.
func slot0Word(t *testing.T, p entity.Pool) common.Hash {
	t.Helper()
	var extra storedSlot0
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	w := new(big.Int).Set(extra.SqrtPriceX96)
	w.Or(w, new(big.Int).Lsh(big.NewInt(extra.Tick.Int64()&0xffffff), 160))
	return common.BigToHash(w)
}

func TestTrack(t *testing.T) {
	t.Parallel()
	fp := fixturePools(t)[0]
	want, _ := hookExtraOf(t, fp.Pool)
	registered := poolConfigRaw{Registered: true, FeeSide: want.FeeSide, Preset: 1, KalosShareBps: want.KalosShareBps}
	track := func(stub *testutil.MulticallStub, chainID valueobject.ChainID, block *big.Int) (Extra, error) {
		h := &Hook{}
		raw, err := h.Track(t.Context(), &uniswapv4.HookParam{
			Cfg:       &uniswapv4.Config{ChainID: chainID},
			RpcClient: stub.Client(), Pool: &fp.Pool, HookAddress: HookAddresses[0], BlockNumber: block,
		})
		if err != nil {
			return Extra{}, err
		}
		var got Extra
		require.NoError(t, json.Unmarshal(raw, &got))
		require.Equal(t, h.Extra, got)
		return got, nil
	}

	// Pinned to the pool's state block.
	stub := trackStub(t, fp.Pool.BlockNumber, fp.Pool, registered, nil)
	got, err := track(stub, valueobject.ChainIDElysium, new(big.Int).SetUint64(fp.Pool.BlockNumber))
	require.NoError(t, err)
	require.Equal(t, want, got)
	targets := stub.Targets()
	require.Len(t, targets, 1)
	require.Equal(t, []common.Address{HookAddresses[0], HookAddresses[0], HookAddresses[0], HookAddresses[0],
		common.HexToAddress("0xE2e105d3F7209A9DA11f83cCf3E7398a753823F1")}, targets[0])

	// A pin at the parent chain's block (Multicall3's block.number on Elysium) is read again at the latest block.
	stub = trackStub(t, fp.Pool.BlockNumber, fp.Pool, registered, nil)
	got, err = track(stub, valueobject.ChainIDElysium, big.NewInt(51_535_956))
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Len(t, stub.Targets(), 1, "the failed pinned read is not a multicall answered")

	// ...but only while the price is the one the tracker stored.
	moved := slot0Word(t, fp.Pool)
	moved[31]++
	stub = trackStub(t, fp.Pool.BlockNumber, fp.Pool, registered, &moved)
	_, err = track(stub, valueobject.ChainIDElysium, big.NewInt(51_535_956))
	require.ErrorIs(t, err, ErrStateMoved)

	// Not registered with the hook, or not on Elysium: an unregistered pool, which refuses quotes.
	stub = trackStub(t, fp.Pool.BlockNumber, fp.Pool, poolConfigRaw{}, nil)
	got, err = track(stub, valueobject.ChainIDElysium, nil)
	require.NoError(t, err)
	require.Equal(t, Extra{}, got)
	stub = trackStub(t, fp.Pool.BlockNumber, fp.Pool, registered, nil)
	got, err = track(stub, valueobject.ChainIDBase, nil)
	require.NoError(t, err)
	require.Equal(t, Extra{}, got)
	require.Empty(t, stub.Targets())

	// A state the contract cannot hold is refused.
	bad := registered
	bad.KalosShareBps = 6000
	stub = trackStub(t, fp.Pool.BlockNumber, fp.Pool, bad, nil)
	_, err = track(stub, valueobject.ChainIDElysium, nil)
	require.ErrorIs(t, err, ErrInvalidFeeState)
}

// The factory registers the hook under its exchange and restores its state from hX.
func TestFactory(t *testing.T) {
	t.Parallel()
	fp := fixturePools(t)[0]
	want, _ := hookExtraOf(t, fp.Pool)
	var extra struct {
		HookExtra json.RawMessage `json:"hX"`
	}
	require.NoError(t, json.Unmarshal([]byte(fp.Pool.Extra), &extra))
	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(extra.HookExtra),
		QuoteAt: 7})
	require.True(t, ok)
	require.Equal(t, string(DexType), hook.GetExchange())
	h := hook.(*Hook)
	require.Equal(t, want, h.Extra)
	require.Equal(t, int64(7), h.quoteAt)
	require.True(t, h.CanBeforeSwap(HookAddresses[0]))
	require.True(t, h.CanAfterSwap(HookAddresses[0]))
}
