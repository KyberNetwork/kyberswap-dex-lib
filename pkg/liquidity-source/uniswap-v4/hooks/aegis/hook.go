package aegis

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	FeeMax = big.NewInt(int64(constants.FeeMax))
)

type Hook struct {
	uniswapv4.Hook
	hook        common.Address
	swapFee     uniswapv4.FeeAmount
	protocolFee *big.Int
}

type AegisExtra struct {
	DynamicFeeManagerAddress common.Address `json:"dFM"`
	PolicyManagerAddress     common.Address `json:"pM"`
	BaseFee                  uint64         `json:"bF"`
	SurgeFee                 uint64         `json:"sF"`
	ManualFee                uint64         `json:"mF"`
	ManualFeeIsSet           bool           `json:"mFS"`
	DynamicFee               uint64         `json:"dF"`
	PoolPOLShare             uint64         `json:"pPS"`
}

type DynamicFeeStateRPC struct {
	BaseFee  *big.Int
	SurgeFee *big.Int
}
type ManualFeeRPC struct {
	ManualFee *big.Int
	IsSet     bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Aegis},
		hook: param.HookAddress,
	}

	if param.HookExtra != "" {
		var extra AegisExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err == nil {
			hook.swapFee = uniswapv4.FeeAmount(extra.DynamicFee)
			hook.protocolFee = big.NewInt(int64(extra.PoolPOLShare))
		}
	}
	return hook
}, HookAddresses...)

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var extra AegisExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return "", err
		}
	}

	if extra.DynamicFeeManagerAddress == (common.Address{}) {
		req := param.RpcClient.NewRequest().SetContext(ctx)
		req.AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: h.hook.Hex(),
			Method: "policyManager",
		}, []any{&extra.PolicyManagerAddress})
		req.AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: h.hook.Hex(),
			Method: "dynamicFeeManager",
		}, []any{&extra.DynamicFeeManagerAddress})
		_, err := req.Aggregate()
		if err != nil {
			return "", err
		}
	}

	req := param.RpcClient.NewRequest().SetContext(ctx)
	var dynamicFeeState DynamicFeeStateRPC
	var manualFee ManualFeeRPC
	var poolPOLShare *big.Int
	req.AddCall(&ethrpc.Call{
		ABI:    aegisDynamicFeeManagerABI,
		Target: extra.DynamicFeeManagerAddress.Hex(),
		Method: "getFeeState",
		Params: []any{eth.StringToBytes32(param.Pool.Address)},
	}, []any{&dynamicFeeState})
	req.AddCall(&ethrpc.Call{
		ABI:    aegisPoolPolicyManagerABI,
		Target: extra.PolicyManagerAddress.Hex(),
		Method: "getManualFee",
		Params: []any{eth.StringToBytes32(param.Pool.Address)},
	}, []any{&manualFee})
	req.AddCall(&ethrpc.Call{
		ABI:    aegisPoolPolicyManagerABI,
		Target: extra.PolicyManagerAddress.Hex(),
		Method: "getPoolPOLShare",
		Params: []any{eth.StringToBytes32(param.Pool.Address)},
	}, []any{&poolPOLShare})
	_, err := req.Aggregate()
	if err != nil {
		return "", err
	}
	extra.BaseFee = dynamicFeeState.BaseFee.Uint64()
	extra.SurgeFee = dynamicFeeState.SurgeFee.Uint64()
	extra.ManualFee = manualFee.ManualFee.Uint64()
	extra.ManualFeeIsSet = manualFee.IsSet
	extra.DynamicFee = lo.Ternary(extra.ManualFeeIsSet, extra.ManualFee, extra.BaseFee+extra.SurgeFee)
	extra.PoolPOLShare = poolPOLShare.Uint64()
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}
	return string(extraBytes), nil
}

func (h *Hook) BeforeSwap(swapHookParams *uniswapv4.BeforeSwapHookParams) (*uniswapv4.BeforeSwapHookResult, error) {
	return &uniswapv4.BeforeSwapHookResult{
		SwapFee: h.swapFee,
		DeltaSpecific: lo.Ternary(swapHookParams.ExactIn, func() *big.Int {
			hookFeeAmt := new(big.Int)
			hookFeeAmt.Mul(swapHookParams.AmountSpecified, big.NewInt(int64(h.swapFee))).Div(hookFeeAmt, FeeMax)
			hookFeeAmt.Mul(hookFeeAmt, h.protocolFee).Div(hookFeeAmt, FeeMax)
			return hookFeeAmt
		}(), new(big.Int),
		),
		DeltaUnSpecific: new(big.Int),
	}, nil
}

func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapHookParams) (hookFeeAmt *big.Int) {
	return lo.Ternary(!swapHookParams.ExactIn, func() *big.Int {
		hookFeeAmt = new(big.Int)
		hookFeeAmt.Mul(swapHookParams.AmountIn, big.NewInt(int64(h.swapFee))).Div(hookFeeAmt, FeeMax)
		hookFeeAmt.Mul(hookFeeAmt, h.protocolFee).Div(hookFeeAmt, FeeMax)
		return hookFeeAmt
	}(), new(big.Int),
	)
}
