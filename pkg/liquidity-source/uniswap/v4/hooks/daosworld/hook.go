// Package daosworld implements DaosWorldLaunchHook, the single Uniswap v4 hook shared by every daos.world launch
// on Robinhood chain (verified on Sourcify: src/launch/DaosWorldLaunchHook.sol). Each pool is native ETH
// (currency0) / token (currency1), LP fee 0, tick spacing 60. Swaps are exact-in only and pay an ETH tax:
//   - buy (zeroForOne): beforeSwap takes floor(ethIn * bips / 10000) off the specified ETH input;
//   - sell: afterSwap takes floor(ethOut * bips / 10000) off the pool's ETH output.
//
// bips decays linearly by block.timestamp from startFeeBips at launchTime to endFeeBips at
// launchTime+decayDuration; all four values are locked once the pool exists.
package daosworld

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var HookAddresses = []common.Address{
	common.HexToAddress("0x4Bc7Ba3a253d048Dc36C17Ac3f4E6484dAD120CC"), // Robinhood
}

var (
	ErrPoolNotRegistered      = errors.New("daosworld: pool not launched through the hook")
	ErrPoolTokensNotAvailable = errors.New("daosworld: pool tokens not available")
)

// NowFn is the clock the decaying tax reads; a variable so tests can pin it.
var NowFn = func() int64 { return time.Now().Unix() }

// Extra mirrors DaosWorldLaunchHook.Launch minus feeRecipient. LaunchTime != 0 marks a tracked pool.
type Extra struct {
	StartFeeBips  uint16 `json:"s,omitempty"`
	EndFeeBips    uint16 `json:"e,omitempty"`
	DecayDuration uint32 `json:"d,omitempty"`
	LaunchTime    uint32 `json:"l,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4DaosWorld}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}, HookAddresses...)

// CloneState returns h itself: the hook's state never changes after launch.
func (h *Hook) CloneState() uniswapv4.Hook { return h }

// Track reads launches(token) once; the rates and launchTime are immutable after the pool is created.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.LaunchTime != 0 {
		return json.Marshal(h.Extra)
	} else if param.Pool == nil || len(param.Pool.Tokens) < 2 {
		return nil, ErrPoolTokensNotAvailable
	}

	var launch struct {
		FeeRecipient  common.Address
		StartFeeBips  uint16
		EndFeeBips    uint16
		DecayDuration uint32
		LaunchTime    uint32
	}
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).AddCall(&ethrpc.Call{
		ABI:    hookABI,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "launches",
		Params: []any{common.HexToAddress(param.Pool.Tokens[1].Address)},
	}, []any{&launch}).Call(); err != nil {
		return nil, err
	}

	// an unregistered or not-yet-launched token keeps an empty Extra, so BeforeSwap refuses to price it
	if launch.FeeRecipient != (common.Address{}) {
		h.Extra = Extra{StartFeeBips: launch.StartFeeBips, EndFeeBips: launch.EndFeeBips,
			DecayDuration: launch.DecayDuration, LaunchTime: launch.LaunchTime}
	}
	return json.Marshal(h.Extra)
}

// feeBips mirrors DaosWorldLaunchHook._feeBips. A clock behind launchTime (a revert on-chain) reads as the
// launch second, the most expensive case.
func (e *Extra) feeBips(now int64) uint64 {
	elapsed := max(now-int64(e.LaunchTime), 0)
	if elapsed >= int64(e.DecayDuration) {
		return uint64(e.EndFeeBips)
	}
	start := uint64(e.StartFeeBips)
	return start - (start-uint64(e.EndFeeBips))*uint64(elapsed)/uint64(e.DecayDuration)
}

// ethTax is the hook's tax on an ETH amount. net=false: amount is pre-tax, tax = floor(amount*bips/10000),
// as on-chain. net=true (CalcIn): amount is what is left after the tax; returns the min tax that leaves it.
func (h *Hook) ethTax(amount *big.Int, net bool) *big.Int {
	var bips big.Int
	bips.SetUint64(h.feeBips(NowFn()))
	if !net {
		return bignumber.MulDivDown(new(big.Int), amount, &bips, bignumber.BasisPoint)
	}
	gross := uniswapv4.GrossBeforeFee(amount, &bips, bignumber.BasisPoint)
	return gross.Sub(gross, amount)
}

// BeforeSwap taxes the ETH leg when it is the specified amount (ZeroForOne == CalcOut): a buy's ETH input
// (CalcOut), or a sell's ETH output (CalcIn, grossed up to what the pool must pay out).
func (h *Hook) BeforeSwap(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.LaunchTime == 0 {
		return nil, ErrPoolNotRegistered
	}
	fee := bignumber.ZeroBI
	if p.ZeroForOne == p.CalcOut {
		fee = h.ethTax(p.AmountSpecified, !p.CalcOut)
	}
	return &uniswapv4.BeforeSwapResult{DeltaSpecified: fee, DeltaUnspecified: bignumber.ZeroBI}, nil
}

// AfterSwap taxes the ETH leg when it is unspecified: a sell's pool ETH output (CalcOut), or a buy's pool ETH
// input (CalcIn, grossed up to what the swapper must send).
func (h *Hook) AfterSwap(p *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.LaunchTime == 0 {
		return nil, ErrPoolNotRegistered
	}
	fee := bignumber.ZeroBI
	if p.ZeroForOne != p.CalcOut {
		fee = h.ethTax(lo.Ternary(p.CalcOut, p.AmountOut, p.AmountIn), !p.CalcOut)
	}
	return &uniswapv4.AfterSwapResult{HookFee: fee}, nil
}
