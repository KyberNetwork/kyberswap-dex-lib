package clanker

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type DynamicFeeHook struct {
	uniswapv4.Hook

	hook            string
	clankerCaller   *ClankerCaller
	poolSim         *uniswapv3.PoolSimulator
	protocolFee     *big.Int
	poolFVars       *PoolDynamicFeeVars
	poolCVars       *PoolDynamicConfigVars
	clankerIsToken0 bool
}

type DynamicFeeExtra struct {
	ProtocolFee     *big.Int
	PoolFVars       *PoolDynamicFeeVars
	PoolCVars       *PoolDynamicConfigVars
	ClankerIsToken0 bool
	ClankerTracked  bool
}

type PoolDynamicConfigVars struct {
	BaseFee                   uint64
	MaxLpFee                  uint64
	ReferenceTickFilterPeriod *uint256.Int
	ResetPeriod               *uint256.Int
	ResetTickFilter           int64
	FeeControlNumerator       *uint256.Int
	DecayFilterBps            *uint256.Int
}

type PoolDynamicFeeVars struct {
	ReferenceTick      int64
	ResetTick          int64
	ResetTickTimestamp *uint256.Int
	LastSwapTimestamp  *uint256.Int
	AppliedVR          uint64
	PrevVA             *uint256.Int
}

type PoolDynamicConfigVarsRPC struct {
	Data struct {
		BaseFee                   *big.Int
		MaxLpFee                  *big.Int
		ReferenceTickFilterPeriod *big.Int
		ResetPeriod               *big.Int
		ResetTickFilter           *big.Int
		FeeControlNumerator       *big.Int
		DecayFilterBps            *big.Int
	}
}

type PoolDynamicFeeVarsRPC struct {
	Data struct {
		ReferenceTick      *big.Int
		ResetTick          *big.Int
		ResetTickTimestamp *big.Int
		LastSwapTimestamp  *big.Int
		AppliedVR          *big.Int
		PrevVA             *big.Int
	}
}

var _ = uniswapv4.RegisterHooksFactory(NewDynamicFeeHook, DynamicFeeHookAddresses...)

func NewDynamicFeeHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &DynamicFeeHook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
		hook: param.HookAddress.Hex(),
	}

	chainID := valueobject.ChainID(param.Cfg.ChainID)

	if param.HookExtra != "" {
		var extra DynamicFeeExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return nil
		}

		hook.clankerIsToken0 = extra.ClankerIsToken0
		hook.protocolFee = extra.ProtocolFee
		hook.poolFVars = extra.PoolFVars
		hook.poolCVars = extra.PoolCVars
	}

	if param.Pool != nil {
		cloned := *param.Pool
		cloned.SwapFee = 0

		hook.poolSim, _ = uniswapv3.NewPoolSimulator(cloned, chainID)
	}

	if param.RpcClient != nil {
		hook.clankerCaller, _ = NewClankerCaller(ClankerAddressByChain[chainID],
			param.RpcClient.GetETHClient())
	}

	return hook
}

func (h *DynamicFeeHook) CloneState() uniswapv4.Hook {
	cloned := *h
	cloned.poolSim = h.poolSim.CloneState().(*uniswapv3.PoolSimulator)
	cloned.protocolFee = new(big.Int).Set(h.protocolFee)

	cloned.poolFVars.ResetTickTimestamp = new(uint256.Int).Set(h.poolFVars.ResetTickTimestamp)
	cloned.poolFVars.LastSwapTimestamp = new(uint256.Int).Set(h.poolFVars.LastSwapTimestamp)
	// cloned.poolFVars.PrevVA = new(uint256.Int).Set(h.poolFVars.PrevVA)

	return &cloned
}

func (h *DynamicFeeHook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var extra DynamicFeeExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err != nil {
			return "", err
		}
	}

	poolBytes := eth.StringToBytes32(param.Pool.Address)

	var (
		poolCVars PoolDynamicConfigVarsRPC
		poolFVars PoolDynamicFeeVarsRPC
	)

	req := param.RpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "protocolFee",
	}, []any{&extra.ProtocolFee})
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "poolConfigVars",
		Params: []any{poolBytes},
	}, []any{&poolCVars})
	req.AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: h.hook,
		Method: "poolFeeVars",
		Params: []any{poolBytes},
	}, []any{&poolFVars})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	extra.PoolCVars = &PoolDynamicConfigVars{
		BaseFee:                   poolCVars.Data.BaseFee.Uint64(),
		MaxLpFee:                  poolCVars.Data.MaxLpFee.Uint64(),
		ResetTickFilter:           poolCVars.Data.ResetTickFilter.Int64(),
		ReferenceTickFilterPeriod: uint256.MustFromBig(poolCVars.Data.ReferenceTickFilterPeriod),
		ResetPeriod:               uint256.MustFromBig(poolCVars.Data.ResetPeriod),
		FeeControlNumerator:       uint256.MustFromBig(poolCVars.Data.FeeControlNumerator),
		DecayFilterBps:            uint256.MustFromBig(poolCVars.Data.DecayFilterBps),
	}

	extra.PoolFVars = &PoolDynamicFeeVars{
		ReferenceTick:      poolFVars.Data.ReferenceTick.Int64(),
		ResetTick:          poolFVars.Data.ResetTick.Int64(),
		AppliedVR:          poolFVars.Data.AppliedVR.Uint64(),
		ResetTickTimestamp: uint256.MustFromBig(poolFVars.Data.ResetTickTimestamp),
		LastSwapTimestamp:  uint256.MustFromBig(poolFVars.Data.LastSwapTimestamp),
		PrevVA:             uint256.MustFromBig(poolFVars.Data.PrevVA),
	}

	if !extra.ClankerTracked {
		if h.clankerCaller == nil {
			return "", ErrClankerCallerIsNil
		}

		token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
		info, err := h.clankerCaller.TokenDeploymentInfo(&bind.CallOpts{Context: ctx}, token0)
		if err != nil {
			return "", err
		}

		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Token.Cmp(token0) == 0
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *DynamicFeeHook) getVolatilityAccumulator(amountIn *big.Int, zeroForOne bool) (uint64, error) {
	tickBefore := int64(h.poolSim.V3Pool.TickCurrent)

	var approxLPFee uint64

	blockTime := uint256.NewInt(uint64(time.Now().Unix()))
	if new(uint256.Int).Add(h.poolFVars.LastSwapTimestamp, h.poolCVars.ReferenceTickFilterPeriod).Lt(blockTime) {
		h.poolFVars.ReferenceTick = tickBefore
		h.poolFVars.ResetTick = tickBefore
		h.poolFVars.ResetTickTimestamp = blockTime

		if new(uint256.Int).Add(h.poolFVars.LastSwapTimestamp, h.poolCVars.ResetPeriod).Gt(blockTime) {
			var appliedVR uint256.Int
			appliedVR.MulDivOverflow(h.poolFVars.PrevVA, h.poolCVars.DecayFilterBps, BPS_DENOMINATOR)

			if appliedVR.GtUint64(maxUint24) {
				h.poolFVars.AppliedVR = maxUint24
			} else {
				h.poolFVars.AppliedVR = appliedVR.Uint64()
			}
		} else {
			h.poolFVars.AppliedVR = 0
		}

		approxLPFee = h.getLpFee(h.poolFVars.AppliedVR)
		// h.setProtocolFee(approxLPFee)
	} else if new(uint256.Int).Add(h.poolFVars.ResetTickTimestamp, h.poolCVars.ResetPeriod).Lt(blockTime) {
		var resetTickDifference int64
		if tickBefore > h.poolFVars.ResetTick {
			resetTickDifference = int64(tickBefore - h.poolFVars.ResetTick)
		} else {
			resetTickDifference = int64(h.poolFVars.ResetTick - tickBefore)
		}

		if resetTickDifference > h.poolCVars.ResetTickFilter {
			// h.poolFVars.ResetTick = tickBefore
			// h.poolFVars.ResetTickTimestamp = blockTime
		} else {
			h.poolFVars.ReferenceTick = tickBefore
			// h.poolFVars.ResetTick = tickBefore
			// h.poolFVars.ResetTickTimestamp = blockTime
			h.poolFVars.AppliedVR = 0

			approxLPFee = h.poolCVars.BaseFee
			// h.setProtocolFee(approxLPFee)
		}
	}

	h.poolFVars.LastSwapTimestamp = blockTime

	// overwrite new LPFee to simulate swap with this swapFee
	h.poolSim.V3Pool.Fee = constants.FeeAmount(approxLPFee)

	tickAfter, err := h.getTicks(amountIn, zeroForOne)
	if err != nil {
		return 0, err
	}

	var tickDifference uint64
	if h.poolFVars.ReferenceTick > tickAfter {
		tickDifference = uint64(h.poolFVars.ReferenceTick - tickAfter)
	} else {
		tickDifference = uint64(tickAfter - h.poolFVars.ReferenceTick)
	}

	volatilityAccumulator := min(tickDifference+h.poolFVars.AppliedVR, maxUint24)

	// h.poolFVars.PrevVA.SetUint64(volatilityAccumulator)

	return volatilityAccumulator, nil
}

func (h *DynamicFeeHook) setProtocolFee(lpFee uint64) {
	h.protocolFee.Mul(big.NewInt(int64(lpFee)), PROTOCOL_FEE_NUMERATOR)
	h.protocolFee.Div(h.protocolFee, FEE_DENOMINATOR)
}

func (h *DynamicFeeHook) getLpFee(volAccumulator uint64) uint64 {
	var fee uint256.Int
	fee.Exp(uint256.NewInt(volAccumulator), u256.U2)

	fee.MulDivOverflow(h.poolCVars.FeeControlNumerator, &fee, FEE_CONTROL_DENOMINATOR)

	fee.AddUint64(&fee, h.poolCVars.BaseFee)

	if fee.GtUint64(h.poolCVars.MaxLpFee) {
		return h.poolCVars.MaxLpFee
	}

	return fee.Uint64()
}

func (h *DynamicFeeHook) BeforeSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int, swapFee uniswapv4.FeeAmount, err error) {
	if h.poolSim == nil {
		return nil, 0, ErrPoolSimIsNil
	}

	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	volAccumulator, err := h.getVolatilityAccumulator(params.AmountIn, params.ZeroForOne)
	if err != nil {
		return nil, 0, err
	}

	lpFee := h.getLpFee(volAccumulator)

	// overwrite protocol fee of hook
	h.setProtocolFee(lpFee)

	// to overwrite swap fee of pool
	swapFee = uniswapv4.FeeAmount(lpFee)

	if !swappingForClanker {
		return big.NewInt(0), swapFee, nil
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	fee.Add(MILLION, h.protocolFee)
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountIn, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &fee, swapFee, nil
}

func (h *DynamicFeeHook) AfterSwap(params *uniswapv4.SwapParam) (hookFeeAmt *big.Int) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if swappingForClanker {
		return big.NewInt(0)
	}

	var delta big.Int
	delta.Mul(params.AmountOut, h.protocolFee)
	delta.Div(&delta, FEE_DENOMINATOR)

	return &delta
}

func (h *DynamicFeeHook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return nil, nil
}

func (h *DynamicFeeHook) simulateSwap(zeroForOne bool, amountIn *big.Int) (uniswapv3.SwapInfo, error) {
	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	fee.Add(MILLION, h.protocolFee)
	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(amountIn, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	amountInForSim := scaledProtocolFee.Add(amountIn, &fee)

	tokenIn, tokenOut := h.poolSim.Pool.GetTokens()[0], h.poolSim.Pool.GetTokens()[1]
	if !zeroForOne {
		tokenIn, tokenOut = tokenOut, tokenIn
	}

	result, err := h.poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInForSim,
		},
		TokenOut: tokenOut,
	})

	if err != nil {
		return uniswapv3.SwapInfo{}, err
	}

	swapInfo := result.SwapInfo.(uniswapv3.SwapInfo)

	return swapInfo, nil
}

func (h *DynamicFeeHook) getTicks(amountIn *big.Int, zeroForOne bool) (int64, error) {
	swapInfo, err := h.simulateSwap(zeroForOne, amountIn)
	if err != nil {
		return 0, err
	}

	return int64(swapInfo.NextStateTickCurrent), nil
}
