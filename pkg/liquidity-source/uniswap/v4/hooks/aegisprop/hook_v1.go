package aegisprop

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	sourcePushFeed    = 1
	sequencerGraceSec = 3600
	pipsDenom         = 1_000_000
	maxInvertTries    = 8
)

// Yield reason codes (integration guide §4.4) that this port can hit.
const (
	reasonUnmanaged   = 60
	reasonPaused      = 61
	reasonStaleOracle = 62
	reasonBadOracle   = 63
	reasonIncomplete  = 64
	reasonSeqDown     = 65
	reasonSeqGrace    = 66
	reasonSeqRequired = 67
	reasonStaleLadder = 68
	reasonInventory   = 69
	reasonTooLarge    = 70
	reasonDepthShort  = 71
	reasonBandZero    = 72
	reasonDust        = 73
)

var (
	uMaxU128    = u256.UMaxU128
	uPipsDenom  = uint256.NewInt(pipsDenom)
	uMaxU127    = new(uint256.Int).Rsh(uMaxU128, 1)
	maxExactOut = new(uint256.Int).Set(uMaxU128)
)

type poolConfigV1RPC struct {
	MmId                       uint16         `abi:"mmId"`
	Token0Decimals             uint8          `abi:"token0Decimals"`
	Token1Decimals             uint8          `abi:"token1Decimals"`
	SlackBps                   uint16         `abi:"slackBps"`
	TakerFeePips               *big.Int       `abi:"takerFeePips"`
	HaircutBpsAtStaleThreshold uint16         `abi:"haircutBpsAtStaleThreshold"`
	FreshThresholdSec          uint32         `abi:"freshThresholdSec"`
	StaleThresholdSec          uint32         `abi:"staleThresholdSec"`
	FreshThresholdSecB         uint32         `abi:"freshThresholdSecB"`
	StaleThresholdSecB         uint32         `abi:"staleThresholdSecB"`
	MultiplierFlags            uint8          `abi:"multiplierFlags"`
	BlackoutSec                uint32         `abi:"blackoutSec"`
	IsVault                    bool           `abi:"isVault"`
	PublishMode                uint8          `abi:"publishMode"`
	FeedA                      common.Address `abi:"feedA"`
	FeedADecimals              uint8          `abi:"feedADecimals"`
	SourceA                    uint8          `abi:"sourceA"`
	RateGasCapA                uint32         `abi:"rateGasCapA"`
	IsCrossRate                bool           `abi:"isCrossRate"`
	InvertDirect               bool           `abi:"invertDirect"`
	RequiresSequencerFeed      bool           `abi:"requiresSequencerFeed"`
	FeedB                      common.Address `abi:"feedB"`
	FeedBDecimals              uint8          `abi:"feedBDecimals"`
	SourceB                    uint8          `abi:"sourceB"`
	RateGasCapB                uint32         `abi:"rateGasCapB"`
}

type roundDataRPC struct {
	RoundId         *big.Int `abi:"roundId"`
	Answer          *big.Int `abi:"answer"`
	StartedAt       *big.Int `abi:"startedAt"`
	UpdatedAt       *big.Int `abi:"updatedAt"`
	AnsweredInRound *big.Int `abi:"answeredInRound"`
}

type ladderRPC struct {
	Header *big.Int `abi:"header"`
	Levels *big.Int `abi:"levels"`
}

type cursorRPC struct {
	BidConsumed *big.Int `abi:"bidConsumed"`
	AskConsumed *big.Int `abi:"askConsumed"`
	BidEpoch    uint64   `abi:"bidEpoch"`
	AskEpoch    uint64   `abi:"askEpoch"`
}

type lane struct{ price, amp uint256.Int }

// HookV1 is the Aegis PropAMM V1 hook: it prices a swap off the tracked ladder, fill cursor, maker
// inventory and push-feed bound exactly as the hook's quote() does (integration guide §5). Pools
// whose oracle is not a single direct push feed are rejected (ErrUnsupported).
type HookV1 struct {
	uniswapv4.Hook `json:"-"`

	Header uint256.Int `json:"hd"` // packed ladder header (§5.2)
	Levels uint256.Int `json:"lw"` // packed ladder lanes

	BidConsumed uint256.Int `json:"bc"`
	AskConsumed uint256.Int `json:"ac"`
	BidEpoch    uint64      `json:"be"`
	AskEpoch    uint64      `json:"ae"`
	Inv0        uint256.Int `json:"i0"`
	Inv1        uint256.Int `json:"i1"`

	MmId           uint16      `json:"m"`
	Paused         bool        `json:"p,omitempty"`
	Unsupported    bool        `json:"u,omitempty"`
	SlackBps       uint64      `json:"s"`
	TakerFeePips   uint64      `json:"f"`
	HaircutAtStale uint64      `json:"h"`
	FreshSec       uint64      `json:"fs"`
	StaleSec       uint64      `json:"ss"`
	Token0Decimals uint8       `json:"d0"`
	Token1Decimals uint8       `json:"d1"`
	HasOracle      bool        `json:"o,omitempty"`
	InvertDirect   bool        `json:"inv,omitempty"`
	FeedADecimals  uint8       `json:"fd,omitempty"`
	FeedAnswer     uint256.Int `json:"fa"` // 0 when the feed answer is not in (0, 2^128)
	FeedUpdatedAt  uint64      `json:"fu"`
	RequiresSeq    bool        `json:"rs,omitempty"`
	SeqFeedSet     bool        `json:"sf,omitempty"`
	SeqDown        bool        `json:"sd,omitempty"`
	SeqStartedAt   uint64      `json:"st,omitempty"`

	bids, asks []lane
	sequence   uint64
	expiration uint64
	anchor     uint256.Int
	quoteAt    int64
}

func (h *HookV1) TracksActivity() {}

func (h *HookV1) AllowEmptyTicks() bool { return true }

func (h *HookV1) now() uint64 {
	if h.quoteAt > 0 {
		return uint64(h.quoteAt)
	}
	return uint64(time.Now().Unix())
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &HookV1{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4AegisProp}}
	_ = param.HookExtra.Unmarshal(hook)
	hook.quoteAt = param.QuoteAt
	hook.decode()
	return hook
}, HookV1Addresses...)

// decode unpacks the ladder words into price/amplitude lanes (§5.2).
func (h *HookV1) decode() {
	var tmp uint256.Int
	hd := &h.Header
	h.anchor.Set(uMaxU128).And(&h.anchor, hd)
	h.sequence = tmp.Rsh(hd, 128).Uint64() & (1<<48 - 1)
	h.expiration = tmp.Rsh(hd, 176).Uint64() & (1<<32 - 1)
	bidLen := int(tmp.Rsh(hd, 242).Uint64() & 7)
	askLen := int(tmp.Rsh(hd, 245).Uint64() & 7)
	ampShift := uint(tmp.Rsh(hd, 248).Uint64() & 0xff)
	h.bids = h.decodeLanes(bidLen, 0, true, ampShift)
	h.asks = h.decodeLanes(askLen, 128, false, ampShift)
}

func (h *HookV1) decodeLanes(n, offset int, isBid bool, ampShift uint) []lane {
	n = min(n, 4)
	lanes := make([]lane, n)
	var delta uint256.Int
	for i := range n {
		word := uint32(new(uint256.Int).Rsh(&h.Levels, uint(offset+32*i)).Uint64())
		priceCode, ampCode := word>>16, word&0xffff

		e, m := priceCode>>12, uint64(priceCode&0xfff)
		f := m
		if e != 0 {
			f = (0x1000 | m) << (e - 1)
		}
		// anchor < 2^128 and f < 2^43 so the product fits in 256 bits.
		delta.Mul(&h.anchor, delta.SetUint64(f))
		delta.AddUint64(&delta, 1<<27-1).Rsh(&delta, 27)
		if isBid {
			lanes[i].price.Sub(&h.anchor, &delta)
		} else {
			lanes[i].price.Add(&h.anchor, &delta)
		}

		e, m = ampCode>>11, uint64(ampCode&0x7ff)
		a := m
		if e != 0 {
			a = (0x800 | m) << (e - 1)
		}
		lanes[i].amp.SetUint64(a)
		lanes[i].amp.Lsh(&lanes[i].amp, ampShift)
	}
	return lanes
}

func (h *HookV1) consumed(zeroForOne bool) *uint256.Int {
	if zeroForOne {
		if h.BidEpoch == h.sequence {
			return &h.BidConsumed
		}
	} else if h.AskEpoch == h.sequence {
		return &h.AskConsumed
	}
	return new(uint256.Int)
}

// walk is the ladder walk of §5.4: it skips the consumed token0 of the side, then fills `amount`
// (token0 when targetIsToken0, else token1) level by level, rounding against the taker.
func walk(lanes []lane, consumed *uint256.Int, targetIsToken0, exactIn bool, amount *uint256.Int,
) (t0, t1, remaining uint256.Int, dust bool) {
	var toSkip, avail0, take0, take1, cap1 uint256.Int
	toSkip.Set(consumed)
	remaining.Set(amount)
	for i := range lanes {
		l := &lanes[i]
		if l.amp.Cmp(&toSkip) <= 0 {
			toSkip.Sub(&toSkip, &l.amp)
			continue
		}
		avail0.Sub(&l.amp, &toSkip)
		toSkip.Clear()
		take0.Clear()
		take1.Clear()
		if targetIsToken0 {
			take0.Set(u256.Min(&remaining, &avail0))
			if exactIn {
				u256.MulDivDown(&take1, &take0, &l.price, uPriceScale)
			} else {
				u256.MulDivUp(&take1, &take0, &l.price, uPriceScale)
			}
			if take1.IsZero() {
				take0.Clear()
			}
		} else {
			if exactIn {
				u256.MulDivUp(&cap1, &avail0, &l.price, uPriceScale)
			} else {
				u256.MulDivDown(&cap1, &avail0, &l.price, uPriceScale)
			}
			if !cap1.IsZero() {
				if remaining.Cmp(&cap1) >= 0 {
					take0.Set(&avail0)
					take1.Set(&cap1)
				} else {
					if exactIn {
						u256.MulDivDown(&take0, &remaining, uPriceScale, &l.price)
					} else {
						u256.MulDivUp(&take0, &remaining, uPriceScale, &l.price)
					}
					if !take0.IsZero() {
						take1.Set(&remaining)
					}
				}
			}
		}
		if take0.IsZero() {
			dust = true
		}
		t0.Add(&t0, &take0)
		t1.Add(&t1, &take1)
		if targetIsToken0 {
			remaining.Sub(&remaining, &take0)
		} else {
			remaining.Sub(&remaining, &take1)
		}
		if remaining.IsZero() {
			break
		}
	}
	return
}

func yield(reason int) error { return fmt.Errorf("%w (reason %d)", ErrYield, reason) }

// mid returns the oracle mid and the staleness haircut in bps (§5.5 step 3, direct push feed).
func (h *HookV1) mid(now uint64) (mid uint256.Int, haircut uint64, err error) {
	if h.FeedUpdatedAt > now+5 || h.FeedAnswer.IsZero() {
		return mid, 0, yield(reasonBadOracle)
	}
	if h.FeedUpdatedAt == 0 {
		return mid, 0, yield(reasonIncomplete)
	}
	var age uint64
	if now > h.FeedUpdatedAt {
		age = now - h.FeedUpdatedAt
	}
	if age > h.StaleSec {
		return mid, 0, yield(reasonStaleOracle)
	}
	if age > h.FreshSec {
		haircut = (h.HaircutAtStale*(age-h.FreshSec) + (h.StaleSec - h.FreshSec) - 1) / (h.StaleSec - h.FreshSec)
	}
	if h.InvertDirect {
		e := int(h.FeedADecimals) + int(h.Token1Decimals) - int(h.Token0Decimals) + 18
		if e < 0 {
			return mid, 0, yield(reasonBadOracle)
		}
		mid.Div(u256.TenPow(e), &h.FeedAnswer)
	} else if e := int(h.Token1Decimals) - int(h.Token0Decimals) + 18 - int(h.FeedADecimals); e >= 0 {
		// answer < 2^128; a product past 2^256 is far above the 2^128 cap below.
		if _, overflow := mid.MulOverflow(&h.FeedAnswer, u256.TenPow(e)); overflow {
			mid.Clear()
		}
	} else {
		mid.Div(&h.FeedAnswer, u256.TenPow(-e))
	}
	if mid.Gt(uMaxU128) {
		mid.Clear()
	}
	if mid.IsZero() {
		return mid, 0, yield(reasonBadOracle)
	}
	return mid, haircut, nil
}

// boundedLeg is §5.5 step 4: the walk, then the oracle bound.
func (h *HookV1) boundedLeg(zeroForOne, exactIn bool, amount, mid *uint256.Int, haircut uint64,
) (t0, t1 uint256.Int, err error) {
	lanes := h.bids
	if !zeroForOne {
		lanes = h.asks
	}
	var remaining uint256.Int
	var dust bool
	t0, t1, remaining, dust = walk(lanes, h.consumed(zeroForOne), zeroForOne == exactIn, exactIn, amount)
	if t0.IsZero() {
		if dust {
			return t0, t1, yield(reasonDust)
		}
		return t0, t1, yield(reasonStaleLadder)
	}
	if !exactIn && !remaining.IsZero() {
		return t0, t1, yield(reasonDepthShort)
	}
	if t0.Gt(uMaxU128) || t1.Gt(uMaxU128) {
		return t0, t1, yield(reasonTooLarge)
	}
	if !h.HasOracle {
		return
	}

	m, tmp := *mid, uint256.Int{}
	if !zeroForOne {
		m.AddUint64(&m, 1) // midUp
	}
	bpsD, up, down := uBpsDenom, uint256.NewInt(bpsDenom+h.SlackBps), uint256.NewInt(bpsDenom-h.SlackBps)
	hairD, hairU := uint256.NewInt(bpsDenom-haircut), uint256.NewInt(bpsDenom+haircut)
	ceiling := func(x *uint256.Int) *uint256.Int {
		u256.MulDivDown(&tmp, x, up, bpsD)
		return u256.MulDivDown(&tmp, &tmp, hairD, bpsD)
	}
	floorOf := func(x *uint256.Int) *uint256.Int {
		u256.MulDivUp(&tmp, x, down, bpsD)
		return u256.MulDivUp(&tmp, &tmp, hairU, bpsD)
	}
	var bound uint256.Int
	switch {
	case exactIn && zeroForOne:
		u256.MulDivDown(&bound, &t0, &m, uPriceScale)
		t1.Set(u256.Min(&t1, ceiling(&bound)))
		if t1.IsZero() {
			return t0, t1, yield(reasonBandZero)
		}
	case exactIn:
		u256.MulDivDown(&bound, &t1, uPriceScale, &m)
		t0.Set(u256.Min(&t0, ceiling(&bound)))
		if t0.IsZero() {
			return t0, t1, yield(reasonBandZero)
		}
	case zeroForOne:
		u256.MulDivUp(&bound, &t1, uPriceScale, &m)
		if f := floorOf(&bound); f.Gt(&t0) {
			t0.Set(f)
		}
	default:
		u256.MulDivUp(&bound, &t0, &m, uPriceScale)
		if f := floorOf(&bound); f.Gt(&t1) {
			t1.Set(f)
		}
	}
	return
}

type fillV1 struct {
	takerIn, takerOut uint256.Int
	t0, t1            uint256.Int // gross legs the cursor and inventory move by
}

// fill is quote() minus the vault steps (§5.5).
func (h *HookV1) fill(zeroForOne, exactIn bool, amount *uint256.Int) (res fillV1, err error) {
	if h.Unsupported {
		return res, ErrUnsupported
	}
	now := h.now()
	if h.MmId == 0 {
		return res, yield(reasonUnmanaged)
	}
	if h.Paused {
		return res, yield(reasonPaused)
	}
	if now >= h.expiration {
		return res, yield(reasonStaleLadder)
	}
	if amount.Gt(uMaxU127) {
		return res, yield(reasonTooLarge)
	}
	if h.RequiresSeq {
		switch {
		case !h.SeqFeedSet:
			return res, yield(reasonSeqRequired)
		case h.SeqStartedAt == 0:
			return res, yield(reasonIncomplete)
		case h.SeqDown:
			return res, yield(reasonSeqDown)
		case now <= h.SeqStartedAt+sequencerGraceSec:
			return res, yield(reasonSeqGrace)
		}
	}
	var mid uint256.Int
	var haircut uint64
	if h.HasOracle {
		if mid, haircut, err = h.mid(now); err != nil {
			return res, err
		}
	}

	t0, t1, err := h.boundedLeg(zeroForOne, exactIn, amount, &mid, haircut)
	if err != nil {
		return res, err
	}
	// Inventory floor (step 6): the maker's balance of the output currency.
	avail, grossOut := &h.Inv0, &t0
	if zeroForOne {
		avail, grossOut = &h.Inv1, &t1
	}
	if grossOut.Gt(avail) {
		if !exactIn || avail.IsZero() {
			return res, yield(reasonInventory)
		}
		if t0, t1, err = h.boundedLeg(zeroForOne, false, avail, &mid, haircut); err != nil {
			return res, err
		}
	}

	grossIn, grossOut := &t0, &t1
	if !zeroForOne {
		grossIn, grossOut = &t1, &t0
	}
	var fee uint256.Int
	if h.TakerFeePips != 0 {
		feeBase := grossIn
		if exactIn {
			feeBase = grossOut
		}
		u256.MulDivUp(&fee, feeBase, uint256.NewInt(h.TakerFeePips), uPipsDenom)
	}
	if exactIn {
		res.takerIn.Set(grossIn)
		res.takerOut.Sub(grossOut, &fee) // fee <= grossOut: pips <= 100_000
	} else {
		res.takerIn.Add(grossIn, &fee)
		res.takerOut.Set(grossOut)
	}
	if res.takerOut.IsZero() {
		return res, yield(reasonDust)
	}
	if exactIn && res.takerIn.Gt(amount) {
		return res, yield(reasonInventory)
	}
	if res.takerIn.Gt(uMaxU127) || res.takerOut.Gt(uMaxU127) {
		return res, yield(reasonTooLarge)
	}
	res.t0.Set(&t0)
	res.t1.Set(&t1)
	return res, nil
}

type SwapInfoV1 struct {
	ZeroForOne bool
	Token0     uint256.Int
	Token1     uint256.Int
}

func (h *HookV1) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if params.AmountSpecified == nil || params.AmountSpecified.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	amt := new(uint256.Int).Set(params.AmountSpecified)

	var f fillV1
	var err error
	if params.CalcOut {
		if f, err = h.fill(params.ZeroForOne, true, amt); err != nil {
			return nil, err
		}
		if f.takerIn.Lt(amt) {
			return nil, ErrPartialFill
		}
	} else if f, err = h.invertExactIn(params.ZeroForOne, amt); err != nil {
		return nil, err
	}

	res := &uniswapv4.BeforeSwapResult{Gas: gasBeforeSwapV1,
		SwapInfo: &SwapInfoV1{ZeroForOne: params.ZeroForOne, Token0: f.t0, Token1: f.t1}}
	if params.CalcOut {
		res.DeltaSpecified = new(uint256.Int).Set(params.AmountSpecified)
		res.DeltaUnspecified = new(uint256.Int).Neg(&f.takerOut)
	} else {
		res.DeltaSpecified = new(uint256.Int).Neg(params.AmountSpecified)
		res.DeltaUnspecified = new(uint256.Int).Set(&f.takerIn)
	}
	return res, nil
}

// invertExactIn finds the exact-in amount whose output covers `out` (swaps execute exact-in, so
// the hook's exact-out branch is only the starting guess). Each retry scales the input by the
// output shortfall.
func (h *HookV1) invertExactIn(zeroForOne bool, out *uint256.Int) (fillV1, error) {
	guess, err := h.fill(zeroForOne, false, out)
	if err != nil {
		return guess, err
	}
	var in, shortfall, next uint256.Int
	in.Set(&guess.takerIn)
	for range maxInvertTries {
		f, err := h.fill(zeroForOne, true, &in)
		if err != nil {
			return f, err
		}
		if f.takerIn.Lt(&in) {
			return f, ErrPartialFill
		}
		if f.takerOut.Cmp(out) >= 0 {
			return f, nil
		}
		shortfall.Sub(out, &f.takerOut)
		u256.MulDivUp(&next, &shortfall, &in, &f.takerOut)
		in.Add(&in, next.AddUint64(&next, 1))
	}
	return fillV1{}, ErrNoInversion
}

func (h *HookV1) CloneState() uniswapv4.Hook {
	cloned := *h // lanes are immutable; the cursor and inventory are values
	return &cloned
}

// UpdateBalance advances the walked side's cursor by the token0 filled and moves the maker's
// inventory: it gains the input leg and pays the output leg.
func (h *HookV1) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfoV1)
	if !ok || info == nil {
		return
	}
	cursor, epoch := &h.AskConsumed, &h.AskEpoch
	if info.ZeroForOne {
		cursor, epoch = &h.BidConsumed, &h.BidEpoch
	}
	cursor.Set(h.consumed(info.ZeroForOne))
	if _, overflow := cursor.AddOverflow(cursor, &info.Token0); overflow || cursor.Gt(uMaxU128) {
		cursor.Set(uMaxU128)
	}
	*epoch = h.sequence

	in, out := &h.Inv1, &h.Inv0
	inAmt, outAmt := &info.Token1, &info.Token0
	if info.ZeroForOne {
		in, out, inAmt, outAmt = &h.Inv0, &h.Inv1, &info.Token0, &info.Token1
	}
	in.Add(in, inAmt)
	if out.Lt(outAmt) {
		out.Clear()
	} else {
		out.Sub(out, outAmt)
	}
}

// GetReserves reports the remaining depth of each side, capped by the maker's inventory.
func (h *HookV1) GetReserves(_ context.Context, _ *uniswapv4.HookParam) (entity.PoolReserves, error) {
	// ask depth (token0): walk all asks as an exact-out of token0; bid depth (token1): exact-in of token0.
	depth0, _, _, _ := walk(h.asks, h.consumed(false), true, false, maxExactOut)
	_, depth1, _, _ := walk(h.bids, h.consumed(true), true, true, maxExactOut)
	r0, r1 := u256.Min(&depth0, &h.Inv0), u256.Min(&depth1, &h.Inv1)
	return entity.PoolReserves{r0.Dec(), r1.Dec()}, nil
}

func (h *HookV1) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	poolId := common.HexToHash(param.Pool.Address)
	hookAddr := hexutil.Encode(param.HookAddress[:])
	newReq := func() *ethrpc.Request {
		req := param.RpcClient.NewRequest().SetContext(ctx)
		if param.BlockNumber != nil {
			req.SetBlockNumber(param.BlockNumber)
		}
		return req
	}
	call := func(method string, params ...any) *ethrpc.Call {
		return &ethrpc.Call{ABI: aegisPropHookV1ABI, Target: hookAddr, Method: method, Params: params}
	}

	var (
		cfg                             poolConfigV1RPC
		ladder                          ladderRPC
		cursor                          cursorRPC
		seqFeed                         common.Address
		emergency, poolPaused, mmPaused bool
	)
	if _, err := newReq().
		AddCall(call("poolConfig", poolId), []any{&cfg}).
		AddCall(call("ladder", poolId), []any{&ladder}).
		AddCall(call("cursor", poolId), []any{&cursor}).
		AddCall(call("sequencerUptimeFeed"), []any{&seqFeed}).
		AddCall(call("emergencyPaused"), []any{&emergency}).
		AddCall(call("poolPaused", poolId), []any{&poolPaused}).
		Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to track aegis propamm v1 state: %w", err)
	}

	var static uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(param.Pool.StaticExtra), &static); err != nil {
		return nil, err
	}
	currency := func(i int) common.Address {
		if static.IsNative[i] {
			return common.Address{}
		}
		return common.HexToAddress(param.Pool.Tokens[i].Address)
	}

	var inv0, inv1 *big.Int
	var feedRound, seqRound roundDataRPC
	req := newReq().
		AddCall(call("mmPaused", cfg.MmId), []any{&mmPaused}).
		AddCall(call("inventory", cfg.MmId, currency(0)), []any{&inv0}).
		AddCall(call("inventory", cfg.MmId, currency(1)), []any{&inv1})
	hasOracle := cfg.SourceA == sourcePushFeed
	if hasOracle {
		req.AddCall(&ethrpc.Call{ABI: aegisPropHookV1ABI, Target: hexutil.Encode(cfg.FeedA[:]),
			Method: "latestRoundData"}, []any{&feedRound})
	}
	hasSeq := seqFeed != (common.Address{})
	if hasSeq {
		req.AddCall(&ethrpc.Call{ABI: aegisPropHookV1ABI, Target: hexutil.Encode(seqFeed[:]),
			Method: "latestRoundData"}, []any{&seqRound})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to track aegis propamm v1 inventory/feeds: %w", err)
	}

	*h = HookV1{
		Hook:           h.Hook,
		MmId:           cfg.MmId,
		Paused:         emergency || poolPaused || mmPaused,
		SlackBps:       uint64(cfg.SlackBps),
		TakerFeePips:   cfg.TakerFeePips.Uint64(),
		HaircutAtStale: uint64(cfg.HaircutBpsAtStaleThreshold),
		FreshSec:       uint64(cfg.FreshThresholdSec),
		StaleSec:       uint64(cfg.StaleThresholdSec),
		Token0Decimals: cfg.Token0Decimals,
		Token1Decimals: cfg.Token1Decimals,
		HasOracle:      hasOracle,
		InvertDirect:   cfg.InvertDirect,
		FeedADecimals:  cfg.FeedADecimals,
		RequiresSeq:    cfg.RequiresSequencerFeed,
		SeqFeedSet:     hasSeq,
		BidEpoch:       cursor.BidEpoch,
		AskEpoch:       cursor.AskEpoch,
		FeedUpdatedAt:  u64(feedRound.UpdatedAt),
		SeqStartedAt:   u64(seqRound.StartedAt),
		SeqDown:        seqRound.Answer != nil && seqRound.Answer.Sign() != 0,
		// Only a single direct push feed (or none) is modelled; vaults, multiplier tokens, cross
		// rates, leg B and other sources quote differently.
		Unsupported: cfg.IsVault || cfg.MultiplierFlags != 0 || cfg.IsCrossRate || cfg.SourceB != 0 ||
			cfg.SourceA > sourcePushFeed,
	}
	h.Header.SetFromBig(ladder.Header)
	h.Levels.SetFromBig(ladder.Levels)
	h.BidConsumed.SetFromBig(cursor.BidConsumed)
	h.AskConsumed.SetFromBig(cursor.AskConsumed)
	h.Inv0.SetFromBig(inv0)
	h.Inv1.SetFromBig(inv1)
	if a := feedRound.Answer; a != nil && a.Sign() > 0 && a.BitLen() <= 128 {
		h.FeedAnswer.SetFromBig(a)
	}
	h.decode()
	return json.Marshal(h)
}

func u64(x *big.Int) uint64 {
	if x == nil || !x.IsUint64() {
		return 0
	}
	return x.Uint64()
}

var _ uniswapv4.Hook = (*HookV1)(nil)
