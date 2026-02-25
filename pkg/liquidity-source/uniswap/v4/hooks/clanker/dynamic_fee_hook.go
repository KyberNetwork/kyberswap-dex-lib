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

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
	ClankerIsToken0 bool `json:",omitempty"`
	ClankerTracked  bool `json:",omitempty"`
}

type PoolDynamicConfigVars struct {
	BaseFee                   uint64 `json:",omitempty"`
	MaxLpFee                  uint64 `json:",omitempty"`
	ReferenceTickFilterPeriod *uint256.Int
	ResetPeriod               *uint256.Int
	ResetTickFilter           int64 `json:",omitempty"`
	FeeControlNumerator       *uint256.Int
	DecayFilterBps            *uint256.Int
}

type PoolDynamicFeeVars struct {
	ReferenceTick      int64 `json:",omitempty"`
	ResetTick          int64 `json:",omitempty"`
	ResetTickTimestamp *uint256.Int
	LastSwapTimestamp  *uint256.Int
	AppliedVR          uint64 `json:",omitempty"`
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

		chainID := valueobject.ChainID(param.Cfg.ChainID)
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
		extra.ClankerIsToken0 = info.Data.Token == token0
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
			appliedVR.MulDivOverflow(h.poolFVars.PrevVA, h.poolCVars.DecayFilterBps, BpsDenominator)

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
	h.protocolFee.Mul(big.NewInt(int64(lpFee)), ProtocolFeeNumerator)
	h.protocolFee.Div(h.protocolFee, FeeDenominator)
}

func (h *DynamicFeeHook) getLpFee(volAccumulator uint64) uint64 {
	var fee uint256.Int
	fee.Exp(uint256.NewInt(volAccumulator), u256.U2)

	fee.MulDivOverflow(h.poolCVars.FeeControlNumerator, &fee, FeeControlDenominator)

	fee.AddUint64(&fee, h.poolCVars.BaseFee)

	if fee.GtUint64(h.poolCVars.MaxLpFee) {
		return h.poolCVars.MaxLpFee
	}

	return fee.Uint64()
}

func (h *DynamicFeeHook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.protocolFee == nil {
		return nil, ErrPoolIsNotTracked
	}

	if h.poolSim == nil {
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

	if params.ZeroForOne == h.clankerIsToken0 {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
			SwapFee:          swapFee,
		}, nil
	}

	var scaledProtocolFee, fee big.Int

	scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)
	// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L297
	fee.Add(Million, h.protocolFee)

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
	if params.ZeroForOne != h.clankerIsToken0 {
		return &uniswapv4.AfterSwapResult{
			HookFee: bignumber.ZeroBI,
		}, nil
	}

	var delta big.Int
	// https://basescan.org/address/0x34a45c6B61876d739400Bd71228CbcbD4F53E8cC#code#F2#L349
	delta.Mul(params.AmountOut, h.protocolFee)
	delta.Div(&delta, FeeDenominator)
	return &uniswapv4.AfterSwapResult{
		HookFee: &delta,
	}, nil
}

func (h *DynamicFeeHook) simulateSwap(amountSpecified *big.Int, zeroForOne bool) (swapInfo uniswapv3.SwapInfo,
	err error) {
	swappingForClanker := zeroForOne != h.clankerIsToken0

	var amountForSim *big.Int

	if !swappingForClanker {
		amountForSim = amountSpecified
	} else {
		var scaledProtocolFee, fee big.Int

		scaledProtocolFee.Mul(h.protocolFee, bignumber.BONE)

		fee.Add(Million, h.protocolFee)

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
