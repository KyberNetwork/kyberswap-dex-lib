// Package flaunch integrates the Flaunch PositionManager family of Uniswap v4 hooks.
//
// Flaunch does not take fees the standard way: the hook holds every pool's liquidity itself and
// captures a percentage of the unspecified token in afterSwap (via afterSwapReturnsDelta), then
// routes it to the coin's creator, referrer and protocol through its own escrow contracts. The
// rate is FeeDistribution.swapFee (1e4 = 100%), 1% protocol-wide by default but overridable per
// pool, so Track reads it from getPoolFeeDistribution instead of hardcoding it.
//
// beforeSwap can additionally fill part of a buy from the hook's InternalSwapPool fee inventory
// at the pool's own TWAP; that fill is priced at roughly the AMM's price and is not modelled.
//
// v1.3+ hooks delegate the fee calculation to a FeeCalculatorDispatcher. Pools routed to a
// SpendGatedSignerFeeCalculator (Game Mode launches) revert in afterSwap unless the swap carries
// a signed authorization, so Track records the gate and the hook refuses to quote while it is
// enforcing; the gate expires on its own at endsAt.
package flaunch

import (
	"context"
	"errors"
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

var (
	ErrGated      = errors.New("flaunch: pool is spend-gated, swaps without a signed authorization revert")
	ErrFeeTooHigh = errors.New("flaunch: swapFee exceeds 100%")

	feeDenom = big.NewInt(FeeDenom)
)

// NowFn is a var so tests can pin the gate-expiry clock.
var NowFn = func() int64 { return time.Now().Unix() }

// Extra is the swap-relevant state Track persists per pool in HookExtra.
type Extra struct {
	// SwapFee is FeeDistribution.swapFee for this pool (FeeDenom = 100%).
	SwapFee uint32 `json:"f"`
	// GateEnabled is true when the pool is routed to a spend-gate calculator with the gate on.
	GateEnabled bool `json:"g,omitempty"`
	// GateEndsAt is the unix time the gate expires; 0 means no expiry.
	GateEndsAt int64 `json:"ge,omitempty"`
}

type Hook struct {
	uniswapv4.Hook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{
		Hook:  &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Flaunch},
		Extra: Extra{SwapFee: DefaultSwapFee},
	}
	if len(param.HookExtra) > 0 {
		var persisted Extra
		// A foreign or empty schema keeps the protocol default until the next Track.
		if err := param.HookExtra.Unmarshal(&persisted); err == nil && persisted.SwapFee != 0 {
			h.Extra = persisted
		}
	}
	return h
}, HookAddresses...)

// feeDistributionRaw is the decode target for getPoolFeeDistribution, field order matching the
// flattened FeeDistribution struct.
type feeDistributionRaw struct {
	SwapFee  *big.Int
	Referrer *big.Int
	Protocol *big.Int
	Active   bool
}

// spendGateSettingsRaw is the decode target for spendGateSettings.
type spendGateSettingsRaw struct {
	Enabled      bool
	WalletCapWei *big.Int
	Settler      common.Address
	EndsAt       *big.Int
}

// Track reads the pool's current swapFee from the hook and, for v1.3+ hooks, follows the
// dispatcher to the pool's fee calculator to learn whether a spend gate is enforcing. Both are
// re-read on every pass: the fee is owner-adjustable and gates are enabled, extended and expired
// over a pool's life. Every call is tolerant of the older hook generations, which have no
// dispatcher (feeCalculator is a different contract or zero) and therefore no gate.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.RpcClient == nil {
		return json.Marshal(h)
	}
	poolId := common.HexToHash(param.Pool.Address)
	hookAddr := hexutil.Encode(param.HookAddress[:])

	var (
		fee        feeDistributionRaw
		calculator common.Address
	)
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{
			ABI: positionManagerABI, Target: hookAddr, Method: "getPoolFeeDistribution", Params: []any{poolId},
		}, []any{&fee}).
		AddCall(&ethrpc.Call{
			ABI: positionManagerABI, Target: hookAddr, Method: "feeCalculator",
		}, []any{&calculator}).
		TryBlockAndAggregate(); err != nil {
		return nil, err
	}
	if fee.SwapFee == nil {
		// getPoolFeeDistribution reverted or the pool is unknown to this hook: keep the last
		// known fee rather than pricing it as fee-free.
		return json.Marshal(h)
	}
	if fee.SwapFee.Cmp(feeDenom) > 0 {
		return nil, ErrFeeTooHigh
	}
	h.SwapFee = uint32(fee.SwapFee.Uint64())
	h.GateEnabled, h.GateEndsAt = false, 0

	// Pre-dispatcher generations stop here: their feeCalculator (if any) has no poolCalculator
	// and the tolerant call below leaves poolCalculator at the zero address.
	if calculator == (common.Address{}) {
		return json.Marshal(h)
	}
	var poolCalculator common.Address
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{
			ABI: feeCalculatorDispatcherABI, Target: hexutil.Encode(calculator[:]), Method: "poolCalculator",
			Params: []any{poolId},
		}, []any{&poolCalculator}).
		TryAggregate(); err != nil {
		return nil, err
	}
	if poolCalculator == (common.Address{}) {
		// Routed to the dispatcher's default calculator, which is stateless and never gates.
		return json.Marshal(h)
	}
	var gate spendGateSettingsRaw
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{
			ABI: spendGatedCalculatorABI, Target: hexutil.Encode(poolCalculator[:]), Method: "spendGateSettings",
			Params: []any{poolId},
		}, []any{&gate}).
		TryAggregate(); err != nil {
		return nil, err
	}
	// A sub-calculator without spendGateSettings (a different IFeeCalculator) leaves EndsAt nil.
	if gate.Enabled && gate.EndsAt != nil {
		h.GateEnabled = true
		if gate.EndsAt.IsInt64() {
			h.GateEndsAt = gate.EndsAt.Int64()
		}
	}
	return json.Marshal(h)
}

// gateEnforcing mirrors SpendGatedSignerFeeCalculator.trackSwap: nothing is enforced once the
// gate is disabled or `block.timestamp > endsAt`.
func (e *Extra) gateEnforcing() bool {
	if !e.GateEnabled {
		return false
	}
	return e.GateEndsAt == 0 || NowFn() <= e.GateEndsAt
}

// BeforeSwap takes no fee (the InternalSwapPool fill is not modelled), but refuses a gated pool
// up front so the simulator never builds a route that would revert.
func (h *Hook) BeforeSwap(_ *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.gateEnforcing() {
		return nil, ErrGated
	}
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          uniswapv4.FeeAmount(0),
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

// AfterSwap mirrors FeeDistributor._captureSwapFees: floor(unspecified * swapFee / 1e4) taken
// from the unspecified currency, so the output on exact-in and the input on exact-out.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.gateEnforcing() {
		return nil, ErrGated
	}
	if h.SwapFee > FeeDenom {
		return nil, ErrFeeTooHigh
	}
	unspecified := params.AmountIn
	if params.CalcOut {
		unspecified = params.AmountOut
	}
	if h.SwapFee == 0 || unspecified == nil || unspecified.Sign() == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(new(big.Int), unspecified, big.NewInt(int64(h.SwapFee)), feeDenom),
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}
