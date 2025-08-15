package clanker

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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

type ClankerDeploymentInfo struct {
	Data struct {
		Token      common.Address
		Hook       common.Address
		Locker     common.Address
		Extensions []common.Address
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
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)

	var (
		poolCVars PoolDynamicConfigVarsRPC
		poolFVars PoolDynamicFeeVarsRPC
		info      ClankerDeploymentInfo
	)

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

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

	if !extra.ClankerTracked {
		req.AddCall(&ethrpc.Call{
			ABI:    clankerABI,
			Target: ClankerAddressByChain[valueobject.ChainID(param.Cfg.ChainID)],
			Method: "tokenDeploymentInfo",
			Params: []any{token0},
		}, []any{&info})
	}

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
		extra.ClankerTracked = true
		extra.ClankerIsToken0 = info.Data.Token.Cmp(token0) == 0
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return "", err
	}

	return string(extraBytes), nil
}

func (h *DynamicFeeHook) getVolatilityAccumulator(amountIn *big.Int, zeroForOne, exactIn bool) (uint64, error) {
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
			resetTickDifference = tickBefore - h.poolFVars.ResetTick
		} else {
			resetTickDifference = h.poolFVars.ResetTick - tickBefore
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

	tickAfter, err := h.getTicks(amountIn, zeroForOne, exactIn)
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

func (h *DynamicFeeHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.poolSim == nil {
		return nil, ErrPoolSimIsNil
	}

	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	volAccumulator, err := h.getVolatilityAccumulator(params.AmountSpecified, params.ZeroForOne, params.ExactIn)
	if err != nil {
		return nil, err
	}

	lpFee := h.getLpFee(volAccumulator)

	// overwrite protocol fee of hook
	h.setProtocolFee(lpFee)

	// to overwrite swap fee of pool
	swapFee := uniswapv4.FeeAmount(lpFee)

	if params.ExactIn && !swappingForClanker || !params.ExactIn && swappingForClanker {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecific:   bignumber.ZeroBI,
			DeltaUnSpecific: bignumber.ZeroBI,
			SwapFee:         swapFee,
		}, nil
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	if params.ExactIn && swappingForClanker {
		// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L297
		fee.Add(MILLION, h.protocolFee)
	} else { // !params.ExactIn && !swappingForClanker
		// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L297
		fee.Sub(MILLION, h.protocolFee)
	}

	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecific:   &fee,
		DeltaUnSpecific: bignumber.ZeroBI,
		SwapFee:         swapFee,
	}, nil
}

func (h *DynamicFeeHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	swappingForClanker := params.ZeroForOne != h.clankerIsToken0

	if params.ExactIn && swappingForClanker || !params.ExactIn && !swappingForClanker {
		return &uniswapv4.AfterSwapResult{
			HookFee: new(big.Int),
			Gas:     0,
		}, nil
	}

	var delta big.Int
	if params.ExactIn && !swappingForClanker {
		// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L349
		delta.Mul(params.AmountOut, h.protocolFee)
	} else { // !params.ExactIn && swappingForClanker
		// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L365
		delta.Mul(params.AmountIn, h.protocolFee)
	}
	delta.Div(&delta, FEE_DENOMINATOR)
	return &uniswapv4.AfterSwapResult{
		HookFee: &delta,
		Gas:     0,
	}, nil
}

func (h *DynamicFeeHook) simulateSwap(amountSpecified *big.Int, zeroForOne, exactIn bool) (swapInfo uniswapv3.SwapInfo, err error) {
	swappingForClanker := zeroForOne != h.clankerIsToken0

	var amountForSim *big.Int

	if exactIn && !swappingForClanker || !exactIn && swappingForClanker {
		amountForSim = amountSpecified
	} else {
		var scaledProtocolFee, fee big.Int

		scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)

		if exactIn {
			fee.Add(MILLION, h.protocolFee)
		} else { // !exactIn && !swappingForClanker
			fee.Sub(MILLION, h.protocolFee)
		}

		scaledProtocolFee.Div(&scaledProtocolFee, &fee)
		fee.Mul(amountSpecified, &scaledProtocolFee)
		fee.Div(&fee, bignumber.BONE)

		amountForSim = new(big.Int).Add(amountSpecified, &fee)
	}

	tokenIn, tokenOut := h.poolSim.Pool.GetTokens()[0], h.poolSim.Pool.GetTokens()[1]
	if !zeroForOne {
		tokenIn, tokenOut = tokenOut, tokenIn
	}

	if exactIn {
		var result *pool.CalcAmountOutResult
		result, err = h.poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  tokenIn,
				Amount: amountForSim,
			},
			TokenOut: tokenOut,
		})
		if err != nil {
			return uniswapv3.SwapInfo{}, err
		}

		swapInfo = result.SwapInfo.(uniswapv3.SwapInfo)

		return
	}

	var result *pool.CalcAmountInResult
	result, err = h.poolSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountForSim,
		},
		TokenIn: tokenIn,
	})
	if err != nil {
		return uniswapv3.SwapInfo{}, err
	}

	swapInfo = result.SwapInfo.(uniswapv3.SwapInfo)

	return
}

func (h *DynamicFeeHook) getTicks(amountSpecified *big.Int, zeroForOne, exactIn bool) (int64, error) {
	swapInfo, err := h.simulateSwap(amountSpecified, zeroForOne, exactIn)
	if err != nil {
		return 0, err
	}

	return int64(swapInfo.NextStateTickCurrent), nil
}
