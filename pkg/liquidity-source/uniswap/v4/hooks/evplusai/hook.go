package evplusai

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	minFee  = 250
	baseFee = 500
	maxFee  = 9000
	// Conservative callback allowances; see the fork comparison in README.md.
	gasBeforeSwap = 15000
	gasAfterSwap  = 45000
)

var (
	HookAddress      = common.HexToAddress("0xcB787A5cDEA8B3715d984d82F1203Fd7bFeBE0c4")
	errInvalidState  = errors.New("evplusai: missing or invalid fee snapshot")
	errInvalidAmount = errors.New("evplusai: invalid swap delta")
)

// Extra is a block-consistent fee snapshot. Timestamp is chain time, never the
// quoting machine's clock. Pool service must refresh on fee updates and expiry.
type Extra struct {
	Fee0For1    uint32 `json:"f01"`
	Fee1For0    uint32 `json:"f10"`
	ExpiresAt   uint64 `json:"expiry"`
	Timestamp   uint64 `json:"ts"`
	ProtocolFee uint32 `json:"pf"`
}

type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	Extra
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4EVPLUSAI}}
	// Leave an invalid snapshot on decode failure, so quotes fail closed while
	// Track can still construct a fresh snapshot for newly discovered pools.
	var extra Extra
	if param.HookExtra.Unmarshal(&extra) == nil && (param.Cfg == nil || param.Cfg.ChainID == 4663) {
		h.Extra = extra
	}
	return h
}, HookAddress)

type slot0 struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	ProtocolFee  *big.Int
	LpFee        *big.Int
}

type feeState struct {
	ZeroForOneFee *big.Int
	OneForZeroFee *big.Int
	ObservedAt    uint64
	ExpiresAt     uint64
	Sequence      uint64
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if param.Cfg == nil || param.Cfg.ChainID != 4663 || param.Pool == nil ||
		param.BlockNumber == nil || param.BlockNumber.Sign() <= 0 || param.RpcClient == nil ||
		param.HookAddress != HookAddress || !common.IsHexAddress(param.Cfg.StateViewAddress) {
		return nil, errInvalidState
	}
	id := common.HexToHash(param.Pool.Address)
	target := hexutil.Encode(param.HookAddress[:])
	var registered bool
	var result struct{ State feeState }
	var slot slot0
	var timestamp *big.Int
	_, err := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: target, Method: "registered", Params: []any{id}}, []any{&registered}).
		AddCall(&ethrpc.Call{ABI: hookABI, Target: target, Method: "feeStateOf", Params: []any{id}}, []any{&result}).
		AddCall(&ethrpc.Call{ABI: stateViewABI, Target: param.Cfg.StateViewAddress, Method: "getSlot0", Params: []any{id}}, []any{&slot}).
		AddCall(&ethrpc.Call{ABI: clockABI, Target: "0xca11bde05977b3631167028862be2a173976ca11", Method: "getCurrentBlockTimestamp"}, []any{&timestamp}).
		Aggregate()
	if err != nil {
		return nil, err
	}
	state := result.State
	if !registered || slot.ProtocolFee == nil || !slot.ProtocolFee.IsUint64() || slot.ProtocolFee.Uint64() > 0xffffff || timestamp == nil || !timestamp.IsUint64() ||
		state.ZeroForOneFee == nil || state.OneForZeroFee == nil ||
		!state.ZeroForOneFee.IsUint64() || !state.OneForZeroFee.IsUint64() ||
		state.ZeroForOneFee.Uint64() > 0xffffff || state.OneForZeroFee.Uint64() > 0xffffff {
		return nil, errInvalidState
	}
	extra := Extra{
		Fee0For1: uint32(state.ZeroForOneFee.Uint64()), Fee1For0: uint32(state.OneForZeroFee.Uint64()),
		ExpiresAt: state.ExpiresAt, Timestamp: timestamp.Uint64(), ProtocolFee: uint32(slot.ProtocolFee.Uint64()),
	}
	if _, err = extra.budget(true); err != nil {
		return nil, err
	}
	if _, err = extra.budget(false); err != nil {
		return nil, err
	}
	return json.Marshal(extra)
}

func (e Extra) budget(zeroForOne bool) (uint64, error) {
	if e.Timestamp == 0 || e.ProtocolFee > 0xffffff || e.ProtocolFee&0xfff > 1000 || e.ProtocolFee>>12 > 1000 {
		return 0, errInvalidState
	}
	if e.Timestamp >= e.ExpiresAt {
		return baseFee, nil
	}
	fee := e.Fee1For0
	if zeroForOne {
		fee = e.Fee0For1
	}
	if fee < minFee || fee > maxFee {
		return 0, errInvalidState
	}
	return uint64(fee), nil
}

// split mirrors the immutable hook's _splitFee. F is the combined nominal
// budget, not the core LP override. All divisions round down as Solidity does.
func split(fee uint64, exactInput bool) (lp, denominator uint64) {
	if exactInput {
		lp = fee * 9 / 10
		return lp, 10 * (1000000 - lp)
	}
	denominator = 10000000 - fee
	return fee * 9 * 1000000 / denominator, denominator
}

func (h *Hook) BeforeSwap(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	fee, err := h.budget(p.ZeroForOne)
	if err != nil {
		return nil, err
	}
	lp, _ := split(fee, p.CalcOut)
	protocol := uint64(h.ProtocolFee >> 12)
	if p.ZeroForOne {
		protocol = uint64(h.ProtocolFee & 0xfff)
	}
	// The parent simulator replaces its total core swap fee with this override,
	// so compose the directional protocol fee here (not in the platform fraction).
	total := protocol + lp - protocol*lp/1000000
	return &uniswapv4.BeforeSwapResult{
		SwapFee: uniswapv4.FeeAmount(total), DeltaSpecified: new(big.Int),
		DeltaUnspecified: new(big.Int), Gas: gasBeforeSwap,
	}, nil
}

func (h *Hook) AfterSwap(p *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	fee, err := h.budget(p.ZeroForOne)
	if err != nil {
		return nil, err
	}
	amount := p.AmountIn
	if p.CalcOut {
		amount = p.AmountOut
	}
	// A core BalanceDelta component is int128. Exact-input output is positive;
	// exact-output input may have magnitude 2^127 (int128 minimum).
	if amount == nil || amount.Sign() < 0 || amount.BitLen() > 128 ||
		(amount.BitLen() == 128 && (p.CalcOut || amount.Cmp(new(big.Int).Lsh(big.NewInt(1), 127)) > 0)) {
		return nil, errInvalidAmount
	}
	_, denominator := split(fee, p.CalcOut)
	var charge, numerator, divisor uint256.Int
	charge.SetFromBig(amount) // bounded above; multiplication cannot overflow uint256
	numerator.SetUint64(fee)
	divisor.SetUint64(denominator)
	charge.Mul(&charge, &numerator).Div(&charge, &divisor)
	return &uniswapv4.AfterSwapResult{HookFee: charge.ToBig(), Gas: gasAfterSwap}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

// Exact core deltas are also the base for the rounded platform charge.
func (h *Hook) UseExactTickTraversal() bool { return true }
