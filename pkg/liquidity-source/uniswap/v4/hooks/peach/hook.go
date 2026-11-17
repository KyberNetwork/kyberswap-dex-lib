package peach

import (
	"time"

	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type MidBlockConfig struct {
	Enabled   bool   `json:"e"`
	Threshold uint32 `json:"t"`
	Fraction  uint32 `json:"f"`
	Cap       uint32 `json:"c"`
}

// Extra is a coherent on-chain snapshot, not a promise of fees in a future transaction.
type Extra struct {
	Tracked         bool           `json:"ok"`
	Timestamp       uint64         `json:"ts"`
	Fee0For1        uint32         `json:"f01"`
	Fee1For0        uint32         `json:"f10"`
	Autonomous0For1 uint32         `json:"a01"`
	Autonomous1For0 uint32         `json:"a10"`
	Floor           uint32         `json:"floor"`
	Cap             uint32         `json:"cap"`
	AbsoluteCap     uint32         `json:"absolute"`
	Poke0For1       uint32         `json:"p01"`
	Poke1For0       uint32         `json:"p10"`
	PokeExpiry      uint64         `json:"exp"`
	PausedUntil     uint64         `json:"pause"`
	PoolPausedUntil uint64         `json:"poolPause"`
	Mid             MidBlockConfig `json:"mid"`
}

type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
	RWA             bool  `json:"-"`
	ValidDeployment bool  `json:"-"`
	QuoteAt         int64 `json:"-"`
	// Route simulations start in a new block, as in hooks/volatile. Only UpdateBalance
	// commits its first pre-swap reference; speculative quotes leave both fields untouched.
	Advanced  bool        `json:"-"`
	OpenPrice uint256.Int `json:"-"`
}

type SwapInfo struct{ OpenPrice uint256.Int }

var NowFn = time.Now

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	d, found := deployments[param.HookAddress]
	h := &Hook{
		BaseHook: uniswapv4.BaseHook{Exchange: DexType}, RWA: d.RWA, QuoteAt: param.QuoteAt,
		ValidDeployment: found && param.Cfg != nil && param.Cfg.ChainID == d.ChainID,
	}
	if err := param.HookExtra.Unmarshal(&h.Extra); err != nil {
		h.Extra = Extra{}
	}
	return h
}, hookAddresses()...)

func (h *Hook) timestamp() uint64 {
	if h.QuoteAt > 0 {
		return uint64(h.QuoteAt)
	}
	return uint64(NowFn().Unix())
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.ValidDeployment {
		return nil, ErrDeployment
	}
	if !h.Tracked {
		return nil, ErrUntracked
	}
	now := h.timestamp()
	if now < h.PausedUntil || now < h.PoolPausedUntil {
		return nil, ErrPaused
	}
	autonomous, poke := h.Autonomous1For0, h.Poke1For0
	if params.ZeroForOne {
		autonomous, poke = h.Autonomous0For1, h.Poke0For1
	}
	fee := resolveFee(autonomous, h.Floor, h.Cap, poke, now < h.PokeExpiry)
	result := &uniswapv4.BeforeSwapResult{
		SwapFee: uniswapv4.FeeAmount(fee), DeltaSpecified: bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI, Gas: gasBeforeSwap,
	}
	if !h.Mid.Enabled {
		return result, nil
	}
	price := params.SqrtPriceX96
	if price == nil || price.IsZero() || price.BitLen() > 160 {
		return nil, ErrInvalidPrice
	}
	if !h.Advanced {
		result.SwapInfo = &SwapInfo{OpenPrice: *price}
		return result, nil
	}
	extra := surcharge(&h.OpenPrice, price, params.ZeroForOne, h.Mid)
	result.SwapFee = uniswapv4.FeeAmount(min(uint64(fee)+uint64(extra), uint64(h.Cap)))
	result.Gas += gasMidBlock
	return result, nil
}

func (h *Hook) UpdateBalance(info any) {
	if state, ok := info.(*SwapInfo); ok && state != nil {
		h.OpenPrice = state.OpenPrice
		h.Advanced = true
	}
}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h // all mutable fields are values, including uint256.Int
	return &clone
}

func (h *Hook) NextTrackAt(*uniswapv4.HookParam) int64 {
	now := h.timestamp()
	var next uint64
	if h.RWA {
		next = max(h.Timestamp+1, now+1)
	}
	for _, at := range []uint64{h.PokeExpiry, h.PausedUntil, h.PoolPausedUntil} {
		if at > now && (next == 0 || at < next) {
			next = at
		}
	}
	return int64(next)
}

var _ uniswapv4.Hook = (*Hook)(nil)
var _ uniswapv4.HookWithSchedule = (*Hook)(nil)
