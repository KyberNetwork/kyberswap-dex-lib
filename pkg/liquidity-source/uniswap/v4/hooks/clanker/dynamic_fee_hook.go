package clanker

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type DynamicFeeHook struct {
	uniswapv4.Hook  `json:"-"`
	*Fork           `json:"-"`
	poolSim         *uniswapv3.PoolSimulator
	ProtocolFee     *big.Int               `json:"p"`
	PoolFVars       *PoolDynamicFeeVars    `json:"f,omitempty"`
	PoolCVars       *PoolDynamicConfigVars `json:"c,omitempty"`
	ClankerIsToken0 bool                   `json:"0,omitempty"`
	ClankerTracked  bool                   `json:"t,omitempty"`
}

type PoolDynamicFeeVars struct {
	ReferenceTick      int64        `json:"r,omitempty"`
	ResetTick          int64        `json:"t,omitempty"`
	ResetTickTimestamp *uint256.Int `json:"s,omitempty"`
	LastSwapTimestamp  *uint256.Int `json:"l,omitempty"`
	AppliedVR          uint64       `json:"a,omitempty"`
	PrevVA             *uint256.Int `json:"p,omitempty"`
}

type PoolDynamicConfigVars struct {
	BaseFee                   uint64       `json:"b,omitempty"`
	MaxLpFee                  uint64       `json:"m,omitempty"`
	ReferenceTickFilterPeriod *uint256.Int `json:"r,omitempty"`
	ResetPeriod               *uint256.Int `json:"p,omitempty"`
	ResetTickFilter           int64        `json:"f,omitempty"`
	FeeControlNumerator       *uint256.Int `json:"c,omitempty"`
	DecayFilterBps            *uint256.Int `json:"d,omitempty"`
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

type TokenDeploymentInfo struct {
	Data struct {
		Token      common.Address
		Hook       common.Address
		Locker     common.Address
		Extensions []common.Address
	}
}

var _ = uniswapv4.RegisterHooksFactory(NewDynamicFeeHook(Clanker), DynamicFeeHookAddresses...)
var _ = uniswapv4.RegisterHooksFactory(NewDynamicFeeHook(Liquid), LiquidDynamicFeeHookAddresses...)

func NewDynamicFeeHook(fork *Fork) func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return func(param *uniswapv4.HookParam) uniswapv4.Hook {
		hook := &DynamicFeeHook{
			Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Clanker},
			Fork: fork,
		}
		_ = param.HookExtra.Unmarshal(&hook)
		if param.Pool != nil {
			cloned := *param.Pool
			cloned.SwapFee = 0
			hook.poolSim, _ = uniswapv3.NewPoolSimulator(cloned, param.Cfg.ChainID)
		}
		return hook
	}
}

func (h *DynamicFeeHook) CloneState() uniswapv4.Hook {
	cloned := *h
	cloned.poolSim = h.poolSim.CloneState().(*uniswapv3.PoolSimulator)
	cloned.ProtocolFee = new(big.Int).Set(h.ProtocolFee)

	cloned.PoolFVars.ResetTickTimestamp = new(uint256.Int).Set(h.PoolFVars.ResetTickTimestamp)
	cloned.PoolFVars.LastSwapTimestamp = new(uint256.Int).Set(h.PoolFVars.LastSwapTimestamp)
	// cloned.PoolFVars.PrevVA = new(uint256.Int).Set(h.PoolFVars.PrevVA)

	return &cloned
}

func (h *DynamicFeeHook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	hook := hexutil.Encode(param.HookAddress[:])
	poolBytes := common.HexToHash(param.Pool.Address)
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)

	var (
		poolCVars PoolDynamicConfigVarsRPC
		poolFVars PoolDynamicFeeVarsRPC
		info      TokenDeploymentInfo
	)

	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: hook,
		Method: "protocolFee",
	}, []any{&h.ProtocolFee}).AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: hook,
		Method: "poolConfigVars",
		Params: []any{poolBytes},
	}, []any{&poolCVars}).AddCall(&ethrpc.Call{
		ABI:    dynamicFeeHookABI,
		Target: hook,
		Method: "poolFeeVars",
		Params: []any{poolBytes},
	}, []any{&poolFVars})
	if !h.ClankerTracked {
		req.AddCall(&ethrpc.Call{
			ABI:    clankerABI,
			Target: h.AddressByChain[param.Cfg.ChainID],
			Method: "tokenDeploymentInfo",
			Params: []any{token0},
		}, []any{&info})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	h.PoolCVars = &PoolDynamicConfigVars{
		BaseFee:                   poolCVars.Data.BaseFee.Uint64(),
		MaxLpFee:                  poolCVars.Data.MaxLpFee.Uint64(),
		ResetTickFilter:           poolCVars.Data.ResetTickFilter.Int64(),
		ReferenceTickFilterPeriod: uint256.MustFromBig(poolCVars.Data.ReferenceTickFilterPeriod),
		ResetPeriod:               uint256.MustFromBig(poolCVars.Data.ResetPeriod),
		FeeControlNumerator:       uint256.MustFromBig(poolCVars.Data.FeeControlNumerator),
		DecayFilterBps:            uint256.MustFromBig(poolCVars.Data.DecayFilterBps),
	}

	h.PoolFVars = &PoolDynamicFeeVars{
		ReferenceTick:      poolFVars.Data.ReferenceTick.Int64(),
		ResetTick:          poolFVars.Data.ResetTick.Int64(),
		AppliedVR:          poolFVars.Data.AppliedVR.Uint64(),
		ResetTickTimestamp: uint256.MustFromBig(poolFVars.Data.ResetTickTimestamp),
		LastSwapTimestamp:  uint256.MustFromBig(poolFVars.Data.LastSwapTimestamp),
		PrevVA:             uint256.MustFromBig(poolFVars.Data.PrevVA),
	}

	if !h.ClankerTracked {
		h.ClankerTracked = true
		h.ClankerIsToken0 = info.Data.Token == token0
	}

	return json.Marshal(h)
}

func (h *DynamicFeeHook) getVolatilityAccumulator(amountIn *big.Int, zeroForOne bool) (uint64, error) {
	tickBefore := int64(h.poolSim.V3Pool.TickCurrent)

	var approxLPFee uint64

	blockTime := uint256.NewInt(uint64(time.Now().Unix()))
	var tmp uint256.Int
	if tmp.Add(h.PoolFVars.LastSwapTimestamp, h.PoolCVars.ReferenceTickFilterPeriod).Lt(blockTime) {
		h.PoolFVars.ReferenceTick = tickBefore
		h.PoolFVars.ResetTick = tickBefore
		h.PoolFVars.ResetTickTimestamp = blockTime

		if tmp.Add(h.PoolFVars.LastSwapTimestamp, h.PoolCVars.ResetPeriod).Gt(blockTime) {
			var appliedVR uint256.Int
			appliedVR.MulDivOverflow(h.PoolFVars.PrevVA, h.PoolCVars.DecayFilterBps, BpsDenominator)

			if appliedVR.GtUint64(maxUint24) {
				h.PoolFVars.AppliedVR = maxUint24
			} else {
				h.PoolFVars.AppliedVR = appliedVR.Uint64()
			}
		} else {
			h.PoolFVars.AppliedVR = 0
		}

		approxLPFee = h.getLpFee(h.PoolFVars.AppliedVR)
		// h.setProtocolFee(approxLPFee)
	} else if tmp.Add(h.PoolFVars.ResetTickTimestamp, h.PoolCVars.ResetPeriod).Lt(blockTime) {
		var resetTickDifference int64
		if tickBefore > h.PoolFVars.ResetTick {
			resetTickDifference = tickBefore - h.PoolFVars.ResetTick
		} else {
			resetTickDifference = h.PoolFVars.ResetTick - tickBefore
		}

		if resetTickDifference > h.PoolCVars.ResetTickFilter {
			// h.PoolFVars.ResetTick = tickBefore
			// h.PoolFVars.ResetTickTimestamp = blockTime
		} else {
			h.PoolFVars.ReferenceTick = tickBefore
			// h.PoolFVars.ResetTick = tickBefore
			// h.PoolFVars.ResetTickTimestamp = blockTime
			h.PoolFVars.AppliedVR = 0

			approxLPFee = h.PoolCVars.BaseFee
			// h.setProtocolFee(approxLPFee)
		}
	}

	h.PoolFVars.LastSwapTimestamp = blockTime

	// overwrite new LPFee to simulate swap with this swapFee
	h.poolSim.V3Pool.Fee = constants.FeeAmount(approxLPFee)

	tickAfter, err := h.getTicks(amountIn, zeroForOne)
	if err != nil {
		return 0, err
	}

	var tickDifference uint64
	if h.PoolFVars.ReferenceTick > tickAfter {
		tickDifference = uint64(h.PoolFVars.ReferenceTick - tickAfter)
	} else {
		tickDifference = uint64(tickAfter - h.PoolFVars.ReferenceTick)
	}

	volatilityAccumulator := min(tickDifference+h.PoolFVars.AppliedVR, maxUint24)

	// h.PoolFVars.PrevVA.SetUint64(volatilityAccumulator)

	return volatilityAccumulator, nil
}

func (h *DynamicFeeHook) setProtocolFee(lpFee uint64) {
	h.ProtocolFee.Mul(big.NewInt(int64(lpFee)), ProtocolFeeNumerator)
	h.ProtocolFee.Div(h.ProtocolFee, FeeDenominator)
}

func (h *DynamicFeeHook) getLpFee(volAccumulator uint64) uint64 {
	var fee uint256.Int
	fee.Exp(fee.SetUint64(volAccumulator), u256.U2)
	fee.MulDivOverflow(h.PoolCVars.FeeControlNumerator, &fee, FeeControlDenominator)
	fee.AddUint64(&fee, h.PoolCVars.BaseFee)

	if fee.GtUint64(h.PoolCVars.MaxLpFee) {
		return h.PoolCVars.MaxLpFee
	}
	return fee.Uint64()
}

func (h *DynamicFeeHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.ProtocolFee == nil {
		return nil, ErrPoolIsNotTracked
	} else if h.poolSim == nil {
		return nil, ErrPoolSimIsNil
	}

	volAccumulator, err := h.getVolatilityAccumulator(params.AmountSpecified, params.ZeroForOne)
	if err != nil {
		return nil, err
	}

	lpFee := h.getLpFee(volAccumulator)

	// overwrite protocol fee of hook
	h.setProtocolFee(lpFee)

	// to overwrite swap fee of pool
	swapFee := uniswapv4.FeeAmount(lpFee)

	if params.ZeroForOne == h.ClankerIsToken0 {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
			SwapFee:          swapFee,
		}, nil
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.ProtocolFee, bignumber.BONE)
	// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L297
	fee.Add(Million, h.ProtocolFee)

	scaledProtocolFee.Div(&scaledProtocolFee, &fee)
	fee.Mul(params.AmountSpecified, &scaledProtocolFee)
	fee.Div(&fee, bignumber.BONE)

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   &fee,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          swapFee,
	}, nil
}

func (h *DynamicFeeHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if params.ZeroForOne != h.ClankerIsToken0 {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	var delta big.Int
	// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L349
	delta.Mul(params.AmountOut, h.ProtocolFee)
	delta.Div(&delta, FeeDenominator)
	return &uniswapv4.AfterSwapResult{
		HookFee: &delta,
	}, nil
}

func (h *DynamicFeeHook) simulateSwap(amountSpecified *big.Int, zeroForOne bool) (swapInfo uniswapv3.SwapInfo,
	err error) {
	swappingForClanker := zeroForOne != h.ClankerIsToken0

	var amountForSim *big.Int

	if !swappingForClanker {
		amountForSim = amountSpecified
	} else {
		var scaledProtocolFee, fee big.Int

		scaledProtocolFee.Mul(h.ProtocolFee, bignumber.BONE)

		fee.Add(Million, h.ProtocolFee)

		scaledProtocolFee.Div(&scaledProtocolFee, &fee)
		fee.Mul(amountSpecified, &scaledProtocolFee)
		fee.Div(&fee, bignumber.BONE)

		amountForSim = new(big.Int).Add(amountSpecified, &fee)
	}

	tokenIn, tokenOut := h.poolSim.GetTokens()[0], h.poolSim.GetTokens()[1]
	if !zeroForOne {
		tokenIn, tokenOut = tokenOut, tokenIn
	}

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

func (h *DynamicFeeHook) getTicks(amountSpecified *big.Int, zeroForOne bool) (int64, error) {
	swapInfo, err := h.simulateSwap(amountSpecified, zeroForOne)
	if err != nil {
		return 0, err
	}

	return int64(swapInfo.NextStateTickCurrent), nil
}
