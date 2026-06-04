package aegis

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	FeeMax = big.NewInt(int64(constants.FeeMax))
)

type Hook struct {
	uniswapv4.Hook           `json:"-"`
	DynamicFeeManagerAddress common.Address `json:"dFM"`
	PolicyManagerAddress     common.Address `json:"pM"`
	DynamicFee               uint64         `json:"dF"`
	ProtocolFee              *big.Int       `json:"pPS"`
}

type DynamicFeeStateRPC struct {
	BaseFee  uint64
	SurgeFee uint64
}

type ManualFeeRPC struct {
	ManualFee uint64
	IsSet     bool
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Aegis}}
	_ = param.HookExtra.Unmarshal(&hook)
	return hook
}, HookAddresses...)

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hook := hexutil.Encode(param.HookAddress[:])
	if valueobject.IsZeroAddress(h.DynamicFeeManagerAddress) {
		if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: hook,
			Method: "policyManager",
		}, []any{&h.PolicyManagerAddress}).AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: hook,
			Method: "dynamicFeeManager",
		}, []any{&h.DynamicFeeManagerAddress}).AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: hook,
			Method: "DYNAMIC_FEE_MANAGER",
		}, []any{&h.DynamicFeeManagerAddress}).TryAggregate(); err != nil {
			return nil, err
		}
	}

	dynamicFeeManager := hexutil.Encode(h.DynamicFeeManagerAddress[:])
	poolId := common.HexToHash(param.Pool.Address)
	var dynamicFeeState DynamicFeeStateRPC
	var manualFee ManualFeeRPC
	var poolPOLShare *big.Int
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber)
	newVersion := valueobject.IsZeroAddress(h.PolicyManagerAddress)
	if newVersion {
		req = req.AddCall(&ethrpc.Call{
			ABI:    aegisDynamicFeeManagerABI,
			Target: dynamicFeeManager,
			Method: "feeQuote",
			Params: []any{poolId},
		}, []any{&h.DynamicFee}).AddCall(&ethrpc.Call{
			ABI:    aegisHookABI,
			Target: hook,
			Method: "hookFeePpm",
			Params: []any{poolId},
		}, []any{&h.ProtocolFee})
	} else { // old unichain version
		policyManager := hexutil.Encode(h.PolicyManagerAddress[:])
		req = req.AddCall(&ethrpc.Call{
			ABI:    aegisDynamicFeeManagerABI,
			Target: dynamicFeeManager,
			Method: "getFeeState",
			Params: []any{poolId},
		}, []any{&dynamicFeeState}).AddCall(&ethrpc.Call{
			ABI:    aegisPoolPolicyManagerABI,
			Target: policyManager,
			Method: "getManualFee",
			Params: []any{poolId},
		}, []any{&manualFee}).AddCall(&ethrpc.Call{
			ABI:    aegisPoolPolicyManagerABI,
			Target: policyManager,
			Method: "getPoolPOLShare",
			Params: []any{poolId},
		}, []any{&poolPOLShare})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	if !newVersion {
		h.DynamicFee = lo.Ternary(manualFee.IsSet, manualFee.ManualFee,
			dynamicFeeState.BaseFee+dynamicFeeState.SurgeFee)
		h.ProtocolFee = poolPOLShare
	}
	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	hookFeeAmt := new(big.Int)
	hookFeeAmt.Mul(params.AmountSpecified, hookFeeAmt.SetUint64(h.DynamicFee)).Div(hookFeeAmt, FeeMax)
	hookFeeAmt.Mul(hookFeeAmt, h.ProtocolFee).Div(hookFeeAmt, FeeMax)
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          uniswapv4.FeeAmount(h.DynamicFee),
		DeltaSpecified:   hookFeeAmt,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(_ *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.ZeroBI,
	}, nil
}
