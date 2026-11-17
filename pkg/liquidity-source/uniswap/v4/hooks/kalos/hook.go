// Package kalos implements KalosDynamic (KalosV4FeeHook), the Uniswap v4 hook of Kalos V4: Kalos's pools on the shared
// PoolManager of Kinetiq Elysium, one pool per pair, every one with the dynamic-fee flag. Each swap pays one total fee
// T in pips, from the pool's preset:
//
//   - a standard pool: a calm minimum per direction plus a variable part from a volatility accumulator in ticks
//     (Trader Joe Liquidity Book v2.1's, held at the burst's peak, with Clanker v4's reset check); directional, so a
//     swap back against the burst pays its minimum except in the same second as the last price move;
//   - a creator pool (one side is a launchpad token naming its creator): a fixed fee.
//
// T is split (splitFee) between the hook's part h and the LP fee lp, both in pips. The hook's part is charged in one
// asset of the pair, the pool's fee asset, rounded up on the gross amount of it:
//
//   - standard pool, fee asset is the swap's input: h = T * kalosShare, lp = (T - h) / (1 - h); beforeSwap takes
//     ceil(amountIn * h) as a specified delta and the pool swaps the rest at lp;
//   - standard pool, fee asset is the output: h = T, lp = 0; afterSwap takes ceil(poolOut * h) from the output as an
//     unspecified delta;
//   - creator pool: h = T, lp = 0, in the token paired with the launchpad token, taken as above (from the input in
//     beforeSwap, or from the output in afterSwap);
//   - no fee asset: lp = T, nothing else.
//
// With the hook's features switched off (featuresOff) or a pool's fee book frozen, a standard pool is a plain v4
// pool at its calm minimum. The tracker reads the pool's config, fee parameters and volatility state; the simulator
// runs the fee engine at the swap's tick and time, so the fee decays between swaps exactly as on chain, and
// UpdateBalance keeps the state the swap leaves. The fee book's split of the LPs' part never changes what the
// swapper pays, so it is not modelled.
package kalos

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// NowFn is the wall clock the fee engine reads when not indexing; a variable so tests can pin it.
var NowFn = func() int64 { return time.Now().Unix() }

var maxInt128 = new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 127), bignumber.One)

// Extra is the hook's tracked state of a pool, read at the pool's state block.
type Extra struct {
	Registered    bool      `json:"r,omitempty"`  // poolConfig(id).registered: created through this hook
	Kind          uint8     `json:"k,omitempty"`  // 0 standard, 1 creator
	FeeSide       uint8     `json:"fs,omitempty"` // 0 none, 1 currency0, 2 currency1: the asset the hook's part is in
	KalosShareBps uint16    `json:"ks,omitempty"` // Kalos's share of a standard pool's fee
	BookFrozen    bool      `json:"bf,omitempty"` // the pool's fee book is frozen: a plain v4 pool
	FeaturesOff   bool      `json:"fo,omitempty"` // featuresOff(): every standard pool is a plain v4 pool
	Params        FeeParams `json:"p"`
	State         FeeState  `json:"s"`
}

// Hook embeds BaseHook by value, not the Hook interface, so a simulator survives the msgpack round trip.
type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
	// quoteAt is the indexing clock (HookParam.QuoteAt); 0 means the wall clock.
	quoteAt int64
}

// SwapInfo carries the volatility state a swap leaves (the hook's write in beforeSwap), and the hook's part of the
// fee and its side, for AfterSwap.
type SwapInfo struct {
	State   FeeState `json:"s"`
	HookFee uint32   `json:"h"`
	FeeIn   bool     `json:"i"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: DexType}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	h.quoteAt = param.QuoteAt
	return h
}, HookAddresses...)

// Track reads poolConfig(id), feeParams(id), feeState(id) and featuresOff() in one multicall, with the PoolManager's
// slot0 word of the pool, pinned to the pool's state block (param.BlockNumber). The volatility state is only right
// with the tick it was left at, so the read must see the price and tick the v4 tracker stored this round (Pool.Extra).
// On Elysium, an Arbitrum Orbit chain, a contract's block.number is the parent chain's block: a multicall that reports
// it (canonical Multicall3; ArbMulticall2 reports the chain's own) pins a block no Elysium node has. A pinned read that
// fails is made again at the latest block, which is kept only if the price is still the stored one (a swap moves it;
// its log tracks the pool again). KalosDynamic only exists on Elysium; on any other chain the pool stays unregistered
// and refuses quotes.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	h.Extra = Extra{}
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDElysium || param.Pool == nil {
		return json.Marshal(h.Extra)
	}
	var stored storedSlot0
	_ = json.Unmarshal([]byte(param.Pool.Extra), &stored)
	manager := uniswapv4.PoolManagerOf(param.Pool, param.Cfg.ChainID)
	if manager == (common.Address{}) && common.IsHexAddress(param.Cfg.PoolManagerAddress) {
		manager = common.HexToAddress(param.Cfg.PoolManagerAddress)
	}

	extra, err := read(ctx, param, param.BlockNumber, manager, &stored)
	if err != nil && param.BlockNumber != nil {
		extra, err = read(ctx, param, nil, manager, &stored)
	}
	if err != nil {
		return nil, err
	}
	h.Extra = extra
	return json.Marshal(h.Extra)
}

// storedSlot0 is the price and tick the v4 tracker read for the pool this round.
type storedSlot0 struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
}

// poolsMappingSlot is the PoolManager's pools mapping slot: a pool's State (slot0 first) is at keccak256(id . 6).
const poolsMappingSlot = 6

func read(ctx context.Context, param *uniswapv4.HookParam, block *big.Int, manager common.Address,
	stored *storedSlot0) (Extra, error) {
	poolID := common.HexToHash(param.Pool.Address)
	hook := hexutil.Encode(param.HookAddress[:])
	var (
		config      poolConfigRaw
		params      feeParamsResult
		state       feeStateResult
		featuresOff bool
		slot0       [32]byte
	)
	req := param.RpcClient.NewRequest().SetContext(ctx).SetOverrides(param.Overrides)
	if block != nil {
		req.SetBlockNumber(block)
	}
	req.AddCall(&ethrpc.Call{ABI: kalosDynamicABI, Target: hook, Method: "poolConfig", Params: []any{poolID}},
		[]any{&config}).
		AddCall(&ethrpc.Call{ABI: kalosDynamicABI, Target: hook, Method: "feeParams", Params: []any{poolID}},
			[]any{&params}).
		AddCall(&ethrpc.Call{ABI: kalosDynamicABI, Target: hook, Method: "feeState", Params: []any{poolID}},
			[]any{&state}).
		AddCall(&ethrpc.Call{ABI: kalosDynamicABI, Target: hook, Method: "featuresOff"}, []any{&featuresOff})
	check := manager != (common.Address{}) && stored.SqrtPriceX96 != nil && stored.Tick != nil
	if check {
		var key [32]byte
		key[31] = poolsMappingSlot
		req.AddCall(&ethrpc.Call{ABI: poolManagerABI, Target: hexutil.Encode(manager[:]), Method: "extsload",
			Params: []any{crypto.Keccak256Hash(poolID[:], key[:])}}, []any{&slot0})
	}
	if _, err := req.Aggregate(); err != nil {
		return Extra{}, err
	}
	if check {
		// slot0: sqrtPriceX96 in bits 0..159, tick (int24) in bits 160..183.
		price := new(big.Int).SetBytes(slot0[12:])
		tick := int64(int32(uint32(slot0[9])<<24|uint32(slot0[10])<<16|uint32(slot0[11])<<8) >> 8)
		if price.Cmp(stored.SqrtPriceX96) != 0 || tick != stored.Tick.Int64() {
			return Extra{}, ErrStateMoved
		}
	}
	if !config.Registered {
		return Extra{}, nil
	}

	extra := Extra{
		Registered:    true,
		Kind:          config.Kind,
		FeeSide:       config.FeeSide,
		KalosShareBps: config.KalosShareBps,
		BookFrozen:    config.BookFrozen,
		FeaturesOff:   featuresOff,
		Params:        params.Params.params(),
		State:         state.State.state(),
	}
	if !extra.valid() {
		return Extra{}, ErrInvalidFeeState
	}
	return extra, nil
}

// valid holds the bounds the contract enforces (_checkFeeParams, _setKalosShare, the pool kinds and fee sides), so
// the simulator never prices a state the chain cannot hold.
func (e *Extra) valid() bool {
	p := &e.Params
	return e.Kind <= kindCreator && e.FeeSide <= feeCurrency1 && e.KalosShareBps <= bps/2 &&
		p.MinFeeZeroForOne >= 100 && p.MinFeeZeroForOne <= p.MaxFeeZeroForOne && p.MaxFeeZeroForOne <= 50_000 &&
		p.MinFeeOneForZero >= 100 && p.MinFeeOneForZero <= p.MaxFeeOneForZero && p.MaxFeeOneForZero <= 50_000 &&
		p.MaxVolatilityAccumulator <= 8_388_607 && p.VariableFeeControl < 1<<24 &&
		abs(e.State.Peak) <= p.MaxVolatilityAccumulator && e.State.VolatilityReference < 1<<24
}

// CloneState copies: every field is a value.
func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

// UpdateBalance keeps the volatility state the swap left, as the hook's beforeSwap writes it.
func (h *Hook) UpdateBalance(swapInfo any) {
	if info, ok := swapInfo.(*SwapInfo); ok && info != nil {
		h.State = info.State
	}
}

// now is the fee engine's clock: the indexing snapshot time when set, else the wall clock read a margin early.
func (h *Hook) now() uint64 {
	if h.quoteAt > 0 {
		return uint64(h.quoteAt)
	}
	return uint64(max(NowFn()-clockMarginSeconds, 0))
}

// plain mirrors _plain: a standard pool with the features off or its fee book frozen swaps at its calm minimum, with
// no hook fee and no change to its volatility state.
func (e *Extra) plain() bool {
	return e.Kind == kindStandard && (e.FeaturesOff || e.BookFrozen)
}

// feeAssetIn is whether the hook's part is charged in the swap's input.
func (e *Extra) feeAssetIn(zeroForOne bool) bool {
	return (e.FeeSide == feeCurrency0) == zeroForOne
}

// BeforeSwap mirrors KalosV4FeeHook._beforeSwap for an exact-input swap: the fee engine moves the pool's volatility
// state and gives the total fee, splitFee its parts. The LP fee is always overridden (lp can be 0). CalcOut: with the
// fee asset as the input, the hook's part comes off the gross input. CalcIn reverses exact-in: with the fee asset as
// the output, the pool's output is grossed up so the afterSwap charge leaves AmountSpecified; with it as the input,
// AfterSwap grosses the input up once the pool's input is known.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Registered {
		return nil, ErrPoolNotRegistered
	}
	if params.AmountSpecified.Cmp(maxInt128) > 0 {
		return nil, ErrAmountTooLarge
	}
	zeroForOne := params.ZeroForOne
	feeIn := h.feeAssetIn(zeroForOne)
	res := &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		Gas:              gasFeeAssetOut,
	}
	if feeIn {
		res.Gas = gasFeeAssetIn
	}

	var hookFee, lpFee uint32
	if h.plain() {
		lpFee = h.Params.calmFee(zeroForOne)
	} else {
		total, _, next, ok := h.Params.computeFee(h.State, int32(params.Tick), h.now(), zeroForOne)
		if !ok {
			return nil, ErrInvalidFeeState
		}
		hookFee, lpFee = splitFee(h.Kind, h.FeeSide, h.KalosShareBps, total, zeroForOne)
		res.SwapInfo = &SwapInfo{State: next, HookFee: hookFee, FeeIn: feeIn}
	}
	res.SwapFee = uniswapv4.FeeAmount(lpFee) | uniswapv4.OverrideFeeFlag

	if hookFee != 0 {
		var amount uint256.Int
		amount.SetFromBig(params.AmountSpecified)
		if params.CalcOut && feeIn {
			fee := hookFeeOf(&amount, hookFee)
			res.DeltaSpecified = fee.ToBig()
		} else if !params.CalcOut && !feeIn {
			gross := grossOf(&amount, hookFee)
			res.DeltaSpecified = gross.Sub(&gross, &amount).ToBig()
		}
	}
	return res, nil
}

// AfterSwap mirrors KalosV4FeeHook._afterSwap for an exact-input swap: with the fee asset as the output, the hook's
// part of the pool's output is returned as the unspecified delta. CalcIn: with the fee asset as the input, the input
// is grossed up so the beforeSwap charge leaves the pool's AmountIn.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	res := &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}
	if params.BeforeSwapResult == nil {
		return res, nil
	}
	info, ok := params.BeforeSwapResult.SwapInfo.(*SwapInfo)
	if !ok || info == nil || info.HookFee == 0 {
		return res, nil
	}
	var amount uint256.Int
	if params.CalcOut && !info.FeeIn {
		amount.SetFromBig(params.AmountOut)
		fee := hookFeeOf(&amount, info.HookFee)
		res.HookFee = fee.ToBig()
	} else if !params.CalcOut && info.FeeIn {
		amount.SetFromBig(params.AmountIn)
		gross := grossOf(&amount, info.HookFee)
		res.HookFee = gross.Sub(&gross, &amount).ToBig()
	}
	return res, nil
}
