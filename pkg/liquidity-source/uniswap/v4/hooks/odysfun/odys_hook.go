package odysfun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook implements the ODYS Elite tax: the pool's own LP fee is always zero, and OdysHook{,2}
// charges a tax on the native-ETH side of every swap (both directions -- ETH is always
// currency0 for Elite pools, see docs.odys.fun "Elite integration"). The rate starts at the
// creator-chosen initialFeeBps and, for a windowed launch, settles permanently to
// settledFeeBps once block.timestamp reaches settleTimestamp.
//
// The contract also runs a short (<=15s) anti-snipe fee ladder right after launch: 15% for the
// first 5s, then a 5% floor until 15s. No verified source exists for either OdysHook or
// OdysHook2, but the same `if elapsed<5 {1500} else if elapsed<15 {max(scheduleFee,500)} else
// {scheduleFee}` shape appears, byte-identical, in independent decompiles of both contracts,
// repeated inline at every one of their several call sites (registerPool, beforeSwap, afterSwap,
// the currentFeeBps getter) -- internally consistent enough, across two separate deployments, to
// treat as settled without an on-chain sub-15s sample (none has been found yet; every real swap
// observed so far landed at elapsed>=18s and matched the schedule with no floor, consistent with
// this shape but not a direct test of it).
type OdysFunHook struct {
	uniswapv4.Hook `json:"-"`

	LaunchTime      uint64 `json:"l,omitempty"`
	InitialFeeBps   uint16 `json:"i,omitempty"`
	SettledFeeBps   uint16 `json:"s,omitempty"`
	SettleTimestamp uint64 `json:"t,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddresses...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &OdysFunHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4OdysFun},
	}
	_ = param.HookExtra.Unmarshal(hook)
	return hook
}

func (h *OdysFunHook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

// hasTaxSchedule reports whether the hook at this address is OdysHook2 (tax schedule, settles
// to a lower rate after a window) as opposed to the older OdysHook (a single static rate
// forever, no settle window, one field short and a different pools() ABI entirely -- calling
// the OdysHook2 ABI against it does not just return zero values, it fails to decode).
func hasTaxSchedule(hookAddress common.Address) bool {
	return hookAddress == HookAddresses[0] // OdysHook2
}

func (h *OdysFunHook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hookAddr := hexutil.Encode(param.HookAddress[:])
	poolID := common.HexToHash(param.Pool.Address)

	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides)

	if hasTaxSchedule(param.HookAddress) {
		var poolInfo struct {
			Token           common.Address
			Currency0       common.Address
			FeeRecipient    common.Address
			LaunchTime      uint64
			Graduated       bool
			InitialFeeBps   uint16
			PlatformAccrued *big.Int
			SettledFeeBps   uint16
			SettleTimestamp uint64
		}
		req.AddCall(&ethrpc.Call{
			ABI:    odysHookABI,
			Target: hookAddr,
			Method: "pools",
			Params: []any{poolID},
		}, []any{&poolInfo})

		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}

		h.LaunchTime = poolInfo.LaunchTime
		h.InitialFeeBps = poolInfo.InitialFeeBps
		h.SettledFeeBps = poolInfo.SettledFeeBps
		h.SettleTimestamp = poolInfo.SettleTimestamp
	} else {
		// OdysHook (first Elite line): pools() returns one word fewer -- no settledFeeBps/
		// settleTimestamp, the rate never changes after launch. Reuse InitialFeeBps as the
		// permanent rate and leave SettleTimestamp at zero so currentFeeBps treats it as "no
		// settle window" (initialFeeBps forever), matching this line's documented behavior.
		var poolInfo struct {
			Token           common.Address
			Currency0       common.Address
			FeeRecipient    common.Address
			LaunchTime      uint64
			Graduated       bool
			InitialFeeBps   uint16
			PlatformAccrued *big.Int
		}
		req.AddCall(&ethrpc.Call{
			ABI:    odysHookLegacyABI,
			Target: hookAddr,
			Method: "pools",
			Params: []any{poolID},
		}, []any{&poolInfo})

		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}

		h.LaunchTime = poolInfo.LaunchTime
		h.InitialFeeBps = poolInfo.InitialFeeBps
		h.SettledFeeBps = 0
		h.SettleTimestamp = 0
	}

	return json.Marshal(h)
}

const (
	// antiSnipeHardWindowSecs/antiSnipeFloorWindowSecs mirror OdysHook2's decompiled
	// beforeSwap/afterSwap anti-snipe ladder (see the doc comment on OdysFunHook for the
	// verification caveat).
	antiSnipeHardWindowSecs  = 5
	antiSnipeFloorWindowSecs = 15
	antiSnipeHardFeeBps      = 1500
	antiSnipeFloorFeeBps     = 500
)

// currentFeeBps mirrors OdysHook2's decompiled current-rate rule:
//   - elapsed < 5s since launch: a flat 15% anti-snipe rate, regardless of the chosen schedule.
//   - 5s <= elapsed < 15s: the schedule's rate (see below), floored at 5% if it would be lower.
//   - elapsed >= 15s: the schedule's rate with no floor -- initialFeeBps forever if no settle
//     window was chosen (settleTimestamp == 0), else settledFeeBps once now reaches
//     settleTimestamp, else still initialFeeBps.
func (h *OdysFunHook) currentFeeBps() uint16 {
	now := uint64(time.Now().Unix())
	elapsed := now - h.LaunchTime // LaunchTime is always in the past by the time Track runs

	if elapsed < antiSnipeHardWindowSecs {
		return antiSnipeHardFeeBps
	}

	fee := h.InitialFeeBps
	if h.SettleTimestamp != 0 && now >= h.SettleTimestamp {
		fee = h.SettledFeeBps
	}

	if elapsed < antiSnipeFloorWindowSecs && fee <= antiSnipeFloorFeeBps {
		return antiSnipeFloorFeeBps
	}
	return fee
}

// feeOnTop returns the tax withheld from a fixed (exactIn-side) ETH amount: fee = amount * bps / 10000.
// Mirrors OdysHook2's internal fee-on-top helper (net vs. gross when amountSpecified <= 0).
func (h *OdysFunHook) feeOnTop(amount *big.Int) *big.Int {
	fee := new(big.Int).Mul(amount, big.NewInt(int64(h.currentFeeBps())))
	return fee.Div(fee, BpsDenominator)
}

// feeGrossUp returns the extra ETH the pool must move so that, after the tax is withheld, the
// user's fixed (exactOut-side) ETH amount is left untouched: fee = amount * bps / (10000 - bps).
// Mirrors OdysHook2's internal gross-up helper (used when amountSpecified > 0).
func (h *OdysFunHook) feeGrossUp(amount *big.Int) *big.Int {
	bps := int64(h.currentFeeBps())
	fee := new(big.Int).Mul(amount, big.NewInt(bps))
	return fee.Div(fee, new(big.Int).Sub(BpsDenominator, big.NewInt(bps)))
}

// specifiedIsETH reports whether BeforeSwapParams.AmountSpecified is ETH-denominated.
// ETH is always currency0 for Elite pools, and AmountSpecified is amountIn for
// CalcAmountOut or amountOut for CalcAmountIn, so it is the ETH leg exactly when:
//   - CalcOut (amountIn specified) and ZeroForOne (ETH in, token out) -- a buy, or
//   - !CalcOut (amountOut specified) and !ZeroForOne (token in, ETH out) -- a sell.
func specifiedIsETH(params *uniswapv4.BeforeSwapParams) bool {
	return params.ZeroForOne == params.CalcOut
}

// BeforeSwap taxes the ETH leg when it is the specified amount (see specifiedIsETH); the
// other leg -- the unspecified amount -- is taxed in AfterSwap instead. CalcOut (dex-lib's
// CalcAmountOut, i.e. a fixed amountIn) mirrors OdysHook2's exactIn branch, which withholds
// the tax on top of that fixed amount; CalcIn (a fixed amountOut) mirrors its exactOut
// branch, which grosses the pool-side amount up so the fixed amount is left untouched.
func (h *OdysFunHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if specifiedIsETH(params) {
		fee := h.feeOnTop(params.AmountSpecified)
		if !params.CalcOut {
			fee = h.feeGrossUp(params.AmountSpecified)
		}
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   fee,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

// AfterSwap taxes the ETH leg when it is the unspecified amount, i.e. whenever BeforeSwap
// did not already tax it: a sell's pool-computed ETH output (CalcOut, AmountOut, exactIn on
// the token side -> fee-on-top) or a buy's pool-computed gross ETH input (CalcIn, AmountIn,
// exactOut on the token side -> gross-up).
func (h *OdysFunHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if specifiedIsETH(params.BeforeSwapParams) {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	if params.CalcOut {
		return &uniswapv4.AfterSwapResult{
			HookFee: h.feeOnTop(params.AmountOut),
		}, nil
	}

	return &uniswapv4.AfterSwapResult{
		HookFee: h.feeGrossUp(params.AmountIn),
	}, nil
}
