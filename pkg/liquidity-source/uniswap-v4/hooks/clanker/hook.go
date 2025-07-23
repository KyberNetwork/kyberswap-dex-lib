package clanker

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	*uniswapv4.BaseHook
	hook  common.Address
	extra string
}

type PoolDynamicFeeVars struct {
	ReferenceTick      *big.Int
	ResetTick          *big.Int
	ResetTickTimestamp *big.Int
	LastSwapTimestamp  *big.Int
	AppliedVR          *big.Int
	PrevVA             *big.Int
}

type PoolDynamicConfigVars struct {
	BaseFee                   *big.Int
	MaxLpFee                  *big.Int
	ReferenceTickFilterPeriod *big.Int
	ResetPeriod               *big.Int
	ResetTickFilter           *big.Int
	FeeControlNumerator       *big.Int
	DecayFilterBps            *big.Int
}

type Extra struct {
	DynamicFeeAddress common.Address `json:"dfa"`
	PoolFeeVars       PoolDynamicFeeVars
	PoolConfigVars    PoolDynamicConfigVars
	TickBefore        *big.Int
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
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
		hook:     param.HookAddress,
		extra:    param.HookExtra,
	}
	return hook
}, HookAddresses...)

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	// var extra Extra
	// if param.HookExtra != "" {
	// 	if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
	// 		return "", err
	// 	}
	// }

	// if extra.DynamicFeeManagerAddress == (common.Address{}) {
	// 	req := param.RpcClient.NewRequest().SetContext(ctx)
	// 	req.AddCall(&ethrpc.Call{
	// 		ABI:    clankerHookABI,
	// 		Target: h.hook.Hex(),
	// 		Method: "policyManager",
	// 	}, []any{&extra.PolicyManagerAddress})
	// 	req.AddCall(&ethrpc.Call{
	// 		ABI:    clankerHookABI,
	// 		Target: h.hook.Hex(),
	// 		Method: "dynamicFeeManager",
	// 	}, []any{&extra.DynamicFeeManagerAddress})
	// 	_, err := req.Aggregate()
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	// req := param.RpcClient.NewRequest().SetContext(ctx)
	// var dynamicFeeState DynamicFeeStateRPC
	// var manualFee ManualFeeRPC
	// var poolPOLShare *big.Int
	// req.AddCall(&ethrpc.Call{
	// 	ABI:    aegisDynamicFeeManagerABI,
	// 	Target: extra.DynamicFeeManagerAddress.Hex(),
	// 	Method: "getFeeState",
	// 	Params: []any{eth.StringToBytes32(param.Pool.Address)},
	// }, []any{&dynamicFeeState})
	// req.AddCall(&ethrpc.Call{
	// 	ABI:    aegisPoolPolicyManagerABI,
	// 	Target: extra.PolicyManagerAddress.Hex(),
	// 	Method: "getManualFee",
	// 	Params: []any{eth.StringToBytes32(param.Pool.Address)},
	// }, []any{&manualFee})
	// req.AddCall(&ethrpc.Call{
	// 	ABI:    aegisPoolPolicyManagerABI,
	// 	Target: extra.PolicyManagerAddress.Hex(),
	// 	Method: "getPoolPOLShare",
	// 	Params: []any{eth.StringToBytes32(param.Pool.Address)},
	// }, []any{&poolPOLShare})
	// _, err := req.Aggregate()
	// if err != nil {
	// 	return "", err
	// }
	// extra.BaseFee = dynamicFeeState.BaseFee.Uint64()
	// extra.SurgeFee = dynamicFeeState.SurgeFee.Uint64()
	// extra.ManualFee = manualFee.ManualFee.Uint64()
	// extra.ManualFeeIsSet = manualFee.IsSet
	// extra.DynamicFee = lo.Ternary(extra.ManualFeeIsSet, extra.ManualFee, extra.BaseFee+extra.SurgeFee)
	// extra.PoolPOLShare = poolPOLShare.Uint64()
	// extraBytes, err := json.Marshal(extra)
	// if err != nil {
	// 	return "", err
	// }
	return "", nil
}

func (h *Hook) BeforeSwap() (hookFeeAmt *big.Int, swapFee uniswapv4.FeeAmount) {
	var extra Extra
	if err := json.Unmarshal([]byte(h.extra), &extra); err != nil {
		return nil, 0
	}
	// return big.NewInt(int64(extra.PoolPOLShare)), uniswapv4.FeeAmount(extra.DynamicFee)

	protocolFee := h.getFee(extra)

	return nil, 0
}

func (h *Hook) AfterSwap() (hookFeeAmt *big.Int) {
	return nil
}

func (h *Hook) getFee(extra Extra) *big.Int {
	volAccumulator := h.getVolatilityAccumulator(extra)

	lpFee := h.getLPFee(volAccumulator, extra.PoolConfigVars.FeeControlNumerator, extra.PoolConfigVars.BaseFee, extra.PoolConfigVars.MaxLpFee)

	protocolFee := h.getProtocolFee(lpFee)

	return protocolFee
}

func (h *Hook) getVolatilityAccumulator(extra Extra) *big.Int {
	poolFVars := extra.PoolFeeVars
	poolCVars := extra.PoolConfigVars

	now := big.NewInt(time.Now().Unix())

	// reset the reference tick if the tick filter period has passed
	if new(big.Int).Add(poolFVars.LastSwapTimestamp, poolCVars.ReferenceTickFilterPeriod).Cmp(now) < 0 {
		// set the reference tick to the tick before the swap
		poolFVars.ReferenceTick = extra.TickBefore

		// set the reset tick to the tick before the swap and record the reset timestamp
		poolFVars.ResetTick = extra.TickBefore
		poolFVars.ResetTickTimestamp = now

		// if the reset period has NOT passed but the tick filter period has, trigger
		// the volatility decay process
		if new(big.Int).Add(poolFVars.LastSwapTimestamp, poolCVars.ResetPeriod).Cmp(now) > 0 {
			var appliedVR big.Int
			appliedVR.Mul(poolFVars.PrevVA, poolCVars.DecayFilterBps)
			appliedVR.Div(&appliedVR, BPS_DENOMINATOR)

			if appliedVR.Cmp(maxUint24) > 0 {
				poolFVars.AppliedVR = new(big.Int).Set(maxUint24)
			} else {
				poolFVars.AppliedVR.Set(&appliedVR)
			}
		} else {
			poolFVars.AppliedVR = big.NewInt(0)
		}

		approxLPFee := h.getLPFee(poolFVars.AppliedVR, poolCVars.FeeControlNumerator, poolCVars.BaseFee, poolCVars.MaxLpFee)

		// set estimated fee for getting simulation closer to actual result
		protocolFee := h.getProtocolFee(approxLPFee)

		_ = protocolFee
	} else if new(big.Int).Add(poolFVars.ResetTickTimestamp, poolCVars.ResetPeriod).Cmp(now) < 0 {
		// check if the tick difference is greater than the reset tick filter
		var resetTickDifference big.Int
		if extra.TickBefore.Cmp(poolFVars.ResetTick) > 0 {
			resetTickDifference.Sub(extra.TickBefore, poolFVars.ResetTick)
		} else {
			resetTickDifference.Sub(poolFVars.ResetTick, extra.TickBefore)
		}

		if resetTickDifference.Cmp(poolCVars.ResetTickFilter) > 0 {
			// the tick difference is large enough, don't kill the reference tick
			poolFVars.ReferenceTick = extra.TickBefore
			poolFVars.ResetTickTimestamp = now
		} else {
			// the tick difference is not large enough, clear the stored volatility
			poolFVars.ReferenceTick = extra.TickBefore
			poolFVars.ResetTick = extra.TickBefore
			poolFVars.ResetTickTimestamp = now
			poolFVars.AppliedVR = big.NewInt(0)

			// clear out fee for simulation
			protocolFee := h.getProtocolFee(poolCVars.BaseFee)

			_ = protocolFee
		}
	}

	poolFVars.LastSwapTimestamp = now

	tickAfter := h.getTicks()

	var tickDifference big.Int
	if poolFVars.ReferenceTick.Cmp(tickAfter) > 0 {
		tickDifference.Sub(poolFVars.ReferenceTick, tickAfter)
	} else {
		tickDifference.Sub(tickAfter, poolFVars.ReferenceTick)
	}

	var volatilityAccumulator big.Int
	volatilityAccumulator.Add(&tickDifference, poolFVars.AppliedVR)

	if volatilityAccumulator.Cmp(maxUint24) > 0 {
		volatilityAccumulator.Set(maxUint24)
	}

	poolFVars.PrevVA.Set(&volatilityAccumulator)

	return &volatilityAccumulator
}

func (h *Hook) getTicks() *big.Int {

}

func (h *Hook) getProtocolFee(lpFee *big.Int) *big.Int {
	var protocolFee big.Int

	protocolFee.Mul(lpFee, PROTOCOL_FEE_NUMERATOR)

	protocolFee.Div(&protocolFee, FEE_DENOMINATOR)

	return &protocolFee
}

func (h *Hook) getLPFee(volAccumulator *big.Int, feeControlNumerator, baseFee, maxLpFee *big.Int) *big.Int {
	var variableFee big.Int

	var volSquared big.Int
	volSquared.Exp(volAccumulator, common.Big2, nil)

	variableFee.Mul(feeControlNumerator, &volSquared)

	variableFee.Div(&variableFee, FEE_CONTROL_DENOMINATOR)

	var fee big.Int
	fee.Add(&variableFee, baseFee)

	if fee.Cmp(maxLpFee) > 0 {
		fee.Set(maxLpFee)
	}

	return fee, nil
}
