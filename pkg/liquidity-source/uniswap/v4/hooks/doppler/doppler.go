package doppler

import (
	"context"
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

type DHook struct { // scheduled
	uniswapv4.Hook `json:"-"`
	IDHook         `json:"-"`
	Asset          common.Address  `json:"a"`
	DHook          common.Address  `json:"h"`
	HookExtra      json.RawMessage `json:"e"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var dHook DHook
	_ = param.HookExtra.Unmarshal(&dHook)
	dHook.Hook = &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Doppler}
	if f := DHooks[dHook.DHook]; f != nil {
		dHook.IDHook = f(dHook.HookExtra)
	}
	return &dHook
}, InitializerAddresses...)

type PoolState struct {
	Numeraire                     common.Address
	TotalTokensOnBondingCurve     *big.Int
	DopplerHook                   common.Address
	GraduationDopplerHookCalldata []byte
	Status                        uint8
	PoolKey                       uniswapv4.PoolKey
	FarTick                       int32
}

func (h *DHook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var poolState, poolState2 PoolState

	req := param.RpcClient.NewRequest().SetContext(ctx)
	paramHook := hexutil.Encode(param.HookAddress[:])
	var err error
	if h.Asset != valueobject.AddrZero {
		_, err = req.AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{h.Asset},
		}, []any{&poolState}).Call()
	} else {
		_, err = req.AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{common.HexToAddress(param.Pool.Tokens[0].Address)},
		}, []any{&poolState}).AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{common.HexToAddress(param.Pool.Tokens[1].Address)},
		}, []any{&poolState2}).TryAggregate()
	}
	if err != nil {
		return nil, err
	}

	if h.Asset == valueobject.AddrZero {
		if poolState.Numeraire == valueobject.AddrZero {
			poolState = poolState2
			h.Asset = poolState.PoolKey.Currency1
		} else {
			h.Asset = poolState.PoolKey.Currency0
		}
	}
	if h.DHook != poolState.DopplerHook {
		h.DHook = poolState.DopplerHook
		if f := DHooks[h.DHook]; f != nil {
			h.IDHook = f(h.HookExtra)
		}
	}

	if h.IDHook != nil {
		if h.HookExtra, err = h.IDHook.Track(ctx, param, h); err != nil {
			return nil, err
		}
	}

	return json.Marshal(h)
}

func (h *DHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.IDHook != nil {
		return h.IDHook.AfterSwap(params, h)
	}
	return h.Hook.AfterSwap(params)
}

type IDHook interface {
	Track(ctx context.Context, param *uniswapv4.HookParam, dExtra *DHook) (json.RawMessage, error)
	AfterSwap(params *uniswapv4.AfterSwapParams, dExtra *DHook) (*uniswapv4.AfterSwapResult, error)
}

// maxSwapFee matches the on-chain MAX_SWAP_FEE = 0.8e6 constant in RehypeTypes.sol.
var maxSwapFee = big.NewInt(800_000)

// RehypeDHook holds the fee schedule for a RehypeDopplerHook pool.
// All fields nil means no fee (fee = 0).
// StartFee only (EndFee/StartingTime/DurationSeconds nil) means static fee = StartFee.
// All four set means a linear decay from StartFee → EndFee over DurationSeconds starting at StartingTime.
type RehypeDHook struct {
	StartFee        int64 `json:"s,omitempty"`
	EndFee          int64 `json:"e,omitempty"`
	StartingTime    int64 `json:"t,omitempty"`
	DurationSeconds int64 `json:"d,omitempty"`
}

func NewRehypeDHook(dExtra json.RawMessage) IDHook {
	var hook RehypeDHook
	_ = json.Unmarshal(dExtra, &hook)
	return &hook
}

// feeScheduleRPC is the ABI decode target for getFeeSchedule.
type feeScheduleRPC struct {
	StartingTime    int64
	StartFee        int64
	EndFee          int64
	LastFee         int64
	DurationSeconds int64
}

type HookFees struct {
	Fees0             *big.Int
	Fees1             *big.Int
	BeneficiaryFees0  *big.Int
	BeneficiaryFees1  *big.Int
	AirlockOwnerFees0 *big.Int
	AirlockOwnerFees1 *big.Int
	CustomFee         int64
}

func (h *RehypeDHook) Track(ctx context.Context, param *uniswapv4.HookParam, dHook *DHook) (json.RawMessage, error) {
	var rpc feeScheduleRPC
	var legacy HookFees
	poolID := common.HexToHash(param.Pool.Address)
	target := hexutil.Encode(dHook.DHook[:])

	_, _ = param.RpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    rehypeDopplerHookABI,
			Target: target,
			Method: "getFeeSchedule",
			Params: []any{poolID},
		}, []any{&rpc}).
		AddCall(&ethrpc.Call{
			ABI:    rehypeDopplerHookABI,
			Target: target,
			Method: "getHookFees",
			Params: []any{poolID},
		}, []any{&legacy}).TryAggregate()

	if rpc.StartFee == 0 {
		// Older contracts without getFeeSchedule: fall back to static customFee.
		if legacy.CustomFee > 0 {
			h.StartFee = legacy.CustomFee
		}
		// else: all 0 → fee 0
	} else if rpc.EndFee == 0 || rpc.EndFee == rpc.StartFee || rpc.DurationSeconds == 0 {
		// No decay — collapse to static.
		h.StartFee = rpc.StartFee
	} else {
		// Decaying schedule: check if already fully decayed at track time.
		startingTime := rpc.StartingTime
		elapsed := time.Now().Unix()
		if startingTime != 0 {
			elapsed -= startingTime
		}
		if elapsed >= rpc.DurationSeconds {
			h.StartFee = rpc.EndFee // already done; store flat
		} else {
			h.StartFee = rpc.StartFee
			h.EndFee = rpc.EndFee
			h.StartingTime = rpc.StartingTime
			h.DurationSeconds = rpc.DurationSeconds
		}
	}

	return json.Marshal(h)
}

// getCurrentFee mirrors the on-chain _getCurrentFee(poolId) logic (without the lastFee shortcut).
func (h *RehypeDHook) getCurrentFee() int64 {
	if h.StartFee == 0 {
		return 0
	}
	// Static fee (no decay fields set).
	if h.DurationSeconds == 0 {
		return h.StartFee
	}

	elapsed := time.Now().Unix() - h.StartingTime
	if elapsed <= 0 {
		return h.StartFee
	} else if elapsed >= h.DurationSeconds {
		return h.EndFee
	}
	// Linear decay: startFee - (startFee - endFee) * elapsed / durationSeconds
	return h.StartFee - (h.StartFee-h.EndFee)*elapsed/h.DurationSeconds
}

func (h *RehypeDHook) AfterSwap(params *uniswapv4.AfterSwapParams, _ *DHook) (*uniswapv4.AfterSwapResult, error) {
	currentFee := h.getCurrentFee()
	if currentFee == 0 {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI, Gas: 198835}, nil
	}
	feeBase := lo.Ternary(params.CalcOut, params.AmountOut, params.AmountIn)
	hookFee := big.NewInt(currentFee)
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.MulDivDown(hookFee, feeBase, hookFee, maxSwapFee),
		Gas:     198835,
	}, nil
}
