package bunniv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/hooklet"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/oracle"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

var _ = uniswapv4.RegisterHooksFactory(NewHook, lo.Keys(HookAddresses)...)

type Hook struct {
	uniswapv4.Hook
	HookExtra
	hook common.Address

	ldf         ldf.ILiquidityDensityFunction
	oracle      *oracle.ObservationStorage
	hooklet     hooklet.IHooklet
	isNative    [2]bool
	tickSpacing int
}

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		hook: param.HookAddress,
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4BunniV2},
	}

	var hookExtra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &hookExtra); err != nil {
			return nil
		}
	}

	if param.Pool != nil {
		if param.Pool.StaticExtra != "" {
			var poolStaticExtra uniswapv4.StaticExtra
			if err := json.Unmarshal([]byte(param.Pool.StaticExtra), &poolStaticExtra); err != nil {
				return nil
			}

			hook.isNative = poolStaticExtra.IsNative
			hook.tickSpacing = int(poolStaticExtra.TickSpacing)
			hook.ldf = InitLDF(hookExtra.LDFAddress, hook.tickSpacing)
		}
	}

	hook.hooklet = InitHooklet(hookExtra.HookletAddress, hookExtra.HookletExtra)
	hook.oracle = oracle.NewObservationStorage(hookExtra.Observations)
	hook.HookExtra = hookExtra

	return hook
}

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	req := param.RpcClient.NewRequest().SetContext(ctx)

	var poolState PoolStateRPC
	req.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: GetHubAddress(h.hook),
		Method: "poolState",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&poolState})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return entity.PoolReserves{
		poolState.Data.Reserve0.Add(poolState.Data.Reserve0, poolState.Data.RawBalance0).String(),
		poolState.Data.Reserve1.Add(poolState.Data.Reserve1, poolState.Data.RawBalance1).String(),
	}, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var hookExtra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &hookExtra); err != nil {
			return "", err
		}
	}

	poolId := eth.StringToBytes32(param.Pool.Address)

	var (
		ldfState  [32]byte
		slot0     Slot0RPC
		poolState PoolStateRPC
		// hookParamsBytes []byte
		storageSlots [5]common.Hash
		topBid       BidRPC
		// nextBid      BidRPC

		poolManagerBalance0, poolManagerBalance1 = big.NewInt(0), big.NewInt(0)
	)

	slotObservationState := crypto.Keccak256Hash(poolId[:], OBSERVATION_STATE_SLOT)
	slotObservationBase := common.BigToHash(new(big.Int).Add(slotObservationState.Big(), bignumber.One))

	slotVaultSharePrices := crypto.Keccak256Hash(poolId[:], VAULT_SHARE_PRICES_SLOT)
	slotCuratorFees := crypto.Keccak256Hash(poolId[:], CURATOR_FEES_SLOT)
	slotHookFee := crypto.Keccak256Hash(poolId[:], HOOK_FEE_SLOT)

	hubAddress := GetHubAddress(h.hook)
	hookAddress := h.hook.Hex()
	token0Address := param.Pool.Tokens[0].Address
	token1Address := param.Pool.Tokens[1].Address
	poolManagerAddress := GetPoolManagerAddress(valueobject.ChainID(param.Cfg.ChainID))

	req1 := param.RpcClient.NewRequest().SetContext(ctx)

	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "extsload",
		Params: []any{
			[]common.Hash{
				slotObservationState,
				slotObservationBase,
				slotVaultSharePrices,
				slotCuratorFees,
				slotHookFee,
			},
		},
	}, []any{&storageSlots})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "getBid",
		Params: []any{poolId, true},
	}, []any{&topBid})
	// req1.AddCall(&ethrpc.Call{
	// 	ABI:    bunniHookABI,
	// 	Target: hookAddress,
	// 	Method: "getBid",
	// 	Params: []any{poolId, false},
	// }, []any{&nextBid})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "ldfStates",
		Params: []any{poolId},
	}, []any{&ldfState})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "slot0s",
		Params: []any{poolId},
	}, []any{&slot0})
	req1.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: hubAddress,
		Method: "poolState",
		Params: []any{poolId},
	}, []any{&poolState})
	// req1.AddCall(&ethrpc.Call{
	// 	ABI:    bunniHubABI,
	// 	Target: hubAddress,
	// 	Method: "hookParams",
	// 	Params: []any{poolId},
	// }, []any{&hookParamsBytes})
	req1.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token0Address,
		Method: "balanceOf",
		Params: []any{poolManagerAddress},
	}, []any{&poolManagerBalance0})
	req1.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token1Address,
		Method: "balanceOf",
		Params: []any{poolManagerAddress},
	}, []any{&poolManagerBalance1})

	res, err := req1.Aggregate()
	if err != nil {
		return "", err
	}

	hookExtra.Slot0 = Slot0{
		SqrtPriceX96:       uint256.MustFromBig(slot0.SqrtPriceX96),
		Tick:               int(slot0.Tick.Int64()),
		LastSwapTimestamp:  slot0.LastSwapTimestamp,
		LastSurgeTimestamp: slot0.LastSurgeTimestamp,
	}

	hookExtra.BunniState = PoolState{
		// LiquidityDensityFunction: poolState.Data.LiquidityDensityFunction,
		// BunniToken:           poolState.Data.BunniToken,
		// Hooklet:              poolState.Data.Hooklet,
		TwapSecondsAgo:       uint32(poolState.Data.TwapSecondsAgo.Int64()),
		LdfParams:            poolState.Data.LdfParams,
		HookParams:           poolState.Data.HookParams,
		LdfType:              poolState.Data.LdfType,
		MinRawTokenRatio0:    uint256.MustFromBig(poolState.Data.MinRawTokenRatio0),
		TargetRawTokenRatio0: uint256.MustFromBig(poolState.Data.TargetRawTokenRatio0),
		MaxRawTokenRatio0:    uint256.MustFromBig(poolState.Data.MaxRawTokenRatio0),
		MinRawTokenRatio1:    uint256.MustFromBig(poolState.Data.MinRawTokenRatio1),
		TargetRawTokenRatio1: uint256.MustFromBig(poolState.Data.TargetRawTokenRatio1),
		MaxRawTokenRatio1:    uint256.MustFromBig(poolState.Data.MaxRawTokenRatio1),
		Currency0Decimals:    poolState.Data.Currency0Decimals,
		Currency1Decimals:    poolState.Data.Currency1Decimals,
		RawBalance0:          uint256.MustFromBig(poolState.Data.RawBalance0),
		RawBalance1:          uint256.MustFromBig(poolState.Data.RawBalance1),
		Reserve0:             uint256.MustFromBig(poolState.Data.Reserve0),
		Reserve1:             uint256.MustFromBig(poolState.Data.Reserve1),
		IdleBalance:          poolState.Data.IdleBalance,
	}

	hookExtra.HookletAddress = poolState.Data.Hooklet
	hookExtra.LDFAddress = poolState.Data.LiquidityDensityFunction
	hookExtra.LdfState = ldfState

	hookExtra.PoolManagerReserves = [2]*uint256.Int{
		uint256.MustFromBig(poolManagerBalance0),
		uint256.MustFromBig(poolManagerBalance1),
	}

	hookExtra.HookParams = decodeHookParams(poolState.Data.HookParams)
	hookExtra.ObservationState = decodeObservationState(storageSlots[0:2])
	hookExtra.VaultSharePrices = decodeVaultSharePrices(storageSlots[2])
	hookExtra.CuratorFees = decodeCuratorFees(storageSlots[3])
	hookExtra.HookFee = decodeHookFee(storageSlots[4])
	hookExtra.AmAmm = decodeAmmPayload(topBid.Data.Manager, topBid.Data.Payload)

	var (
		redeemRates [2]*big.Int
		maxDeposits [2]*big.Int
	)
	req2 := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	for i, vault := range []string{poolState.Data.Vault0.Hex(), poolState.Data.Vault1.Hex()} {
		req2.AddCall(&ethrpc.Call{
			ABI:    erc4626ABI,
			Target: vault,
			Method: "previewRedeem",
			Params: []any{WAD.ToBig()},
		}, []any{&redeemRates[i]})
		req2.AddCall(&ethrpc.Call{
			ABI:    erc4626ABI,
			Target: poolState.Data.Vault1.Hex(),
			Method: "maxDeposit",
			Params: []any{common.HexToAddress(hubAddress)},
		}, []any{&maxDeposits[i]})
	}

	var slotObservations = make([]common.Hash, 0, hookExtra.ObservationState.CardinalityNext)
	for i := range hookExtra.ObservationState.CardinalityNext {
		slotObservations = append(slotObservations,
			common.BigToHash(big.NewInt(int64(6+i))))
	}

	var observationHashes = make([]common.Hash, len(slotObservations))
	req2.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: h.hook.Hex(),
		Method: "extsload",
		Params: []any{slotObservations},
	}, []any{&observationHashes})

	if _, err := req2.TryBlockAndAggregate(); err != nil {
		return "", err
	}

	hookExtra.Vaults = [2]Vault{
		{
			Address:    poolState.Data.Vault0,
			Decimals:   poolState.Data.Vault0Decimals,
			RedeemRate: uint256.MustFromBig(redeemRates[0]),
			MaxDeposit: uint256.MustFromBig(maxDeposits[0]),
		},
		{
			Address:    poolState.Data.Vault1,
			Decimals:   poolState.Data.Vault1Decimals,
			RedeemRate: uint256.MustFromBig(redeemRates[1]),
			MaxDeposit: uint256.MustFromBig(maxDeposits[1]),
		},
	}

	hookExtra.Observations = decodeObservations(observationHashes)

	hookletExtra, err := h.hooklet.Track(ctx, hooklet.HookletParams{
		RpcClient:      param.RpcClient,
		HookletAddress: hookExtra.HookletAddress,
		HookletExtra:   hookExtra.HookletExtra,
		PoolId:         poolId,
	})
	if err != nil {
		return "", err
	}

	hookExtra.HookletExtra = hookletExtra

	newHookExtra, err := json.Marshal(&hookExtra)
	if err != nil {
		return "", err
	}

	return string(newHookExtra), nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapHookParams) (*uniswapv4.BeforeSwapHookResult, error) {
	if h.ldf == nil {
		return nil, fmt.Errorf("ldf is not initialized")
	}

	amountSpecified := uint256.MustFromBig(params.AmountSpecified)

	blockTimestamp := uint32(time.Now().Unix())

	feeOverridden, feeOverride, priceOverridden, sqrtPriceX96Override := h.hooklet.BeforeSwap(&hooklet.SwapParams{
		ZeroForOne: params.ZeroForOne,
	})

	if priceOverridden {
		h.Slot0.SqrtPriceX96.Set(sqrtPriceX96Override)
		var err error
		h.Slot0.Tick, err = math.GetTickAtSqrtPrice(sqrtPriceX96Override)
		if err != nil {
			return nil, err
		}
	}

	sqrtPriceLimitX96 := lo.Ternary(params.ZeroForOne,
		new(uint256.Int).AddUint64(v3Utils.MinSqrtRatioU256, 1),
		new(uint256.Int).SubUint64(v3Utils.MaxSqrtRatioU256, 1),
	)

	if h.Slot0.SqrtPriceX96.IsZero() ||
		(params.ZeroForOne && sqrtPriceLimitX96.Cmp(h.Slot0.SqrtPriceX96) >= 0) ||
		(!params.ZeroForOne && sqrtPriceLimitX96.Cmp(h.Slot0.SqrtPriceX96) <= 0) ||
		params.AmountSpecified.Cmp(bignumber.MAX_INT_128) > 0 ||
		params.AmountSpecified.Cmp(bignumber.MIN_INT_128) < 0 {

		return nil, fmt.Errorf("BunniHook__InvalidSwap")
	}

	var balance0 uint256.Int
	balance0.Add(h.BunniState.RawBalance0, getReservesInUnderlying(h.Vaults[0], h.BunniState.Reserve0))

	var balance1 uint256.Int
	balance1.Add(h.BunniState.RawBalance1, getReservesInUnderlying(h.Vaults[1], h.BunniState.Reserve1))

	h.updateOracle(blockTimestamp)

	useLDFTwap := h.BunniState.TwapSecondsAgo != 0
	useFeeTwap := !feeOverridden && h.HookParams.FeeTwapSecondsAgo != 0

	var arithmeticMeanTick int64
	var feeMeanTick int64
	var err error
	if useLDFTwap && useFeeTwap {
		tickCumulatives, err := h.oracle.ObserveTriple(
			h.ObservationState.IntermediateObservation,
			blockTimestamp,
			[]uint32{0, h.BunniState.TwapSecondsAgo, h.HookParams.FeeTwapSecondsAgo},
			h.Slot0.Tick,
			h.ObservationState.Index,
			h.ObservationState.Cardinality,
		)
		if err != nil {
			return nil, err
		}

		arithmeticMeanTick = (tickCumulatives[0] - tickCumulatives[1]) / int64(h.BunniState.TwapSecondsAgo)
		feeMeanTick = (tickCumulatives[0] - tickCumulatives[2]) / int64(h.HookParams.FeeTwapSecondsAgo)
	} else if useLDFTwap {
		arithmeticMeanTick, err = h.getTwap(blockTimestamp)
		if err != nil {
			return nil, err
		}
	} else if useFeeTwap {
		feeMeanTick, err = h.getTwap(blockTimestamp)
		if err != nil {
			return nil, err
		}
	}

	totalLiquidity, liquidityDensityOfRoundedTickX96, currentActiveBalance0, currentActiveBalance1,
		newLdfState, shouldSurge, err := h.queryLDF(h.Slot0.SqrtPriceX96, h.Slot0.Tick,
		int(arithmeticMeanTick), h.LdfState, &balance0, &balance1, h.BunniState.IdleBalance)
	if err != nil {
		return nil, err
	}

	if (params.ZeroForOne && currentActiveBalance1.IsZero()) ||
		(!params.ZeroForOne && currentActiveBalance0.IsZero()) ||
		totalLiquidity.IsZero() ||
		(!params.ExactIn &&
			lo.Ternary(params.ZeroForOne, currentActiveBalance1, currentActiveBalance0).Lt(amountSpecified)) {
		return nil, fmt.Errorf("BunniHook__RequestedOutputExceedsBalance")
	}

	shouldSurge = shouldSurge && h.BunniState.LdfType != STATIC

	if h.BunniState.LdfType == DYNAMIC_AND_STATEFUL {
		h.LdfState = newLdfState
	}

	if shouldSurge {
		newIdleBalance, err := h.computeIdleBalance(currentActiveBalance0, currentActiveBalance1, &balance0, &balance1)
		if err != nil {
			return nil, err
		}

		h.BunniState.IdleBalance = newIdleBalance
	}

	shouldSurgeFromVaults, err := h.shouldSurgeFromVaults()
	if err != nil {
		return nil, err
	}

	shouldSurge = shouldSurge || shouldSurgeFromVaults

	updatedSqrtPriceX96, updatedTick, inputAmount, outputAmount, err := h.computeSwap(BunniComputeSwapInput{
		TotalLiquidity:                   totalLiquidity,
		LiquidityDensityOfRoundedTickX96: liquidityDensityOfRoundedTickX96,
		CurrentActiveBalance0:            currentActiveBalance0,
		CurrentActiveBalance1:            currentActiveBalance1,
		ArithmeticMeanTick:               int(arithmeticMeanTick),
		ZeroForOne:                       params.ZeroForOne,
		ExactIn:                          params.ExactIn,
		AmountSpecified:                  amountSpecified,
		SqrtPriceLimitX96:                sqrtPriceLimitX96,
		LdfState:                         newLdfState,
	})
	if err != nil {
		return nil, err
	}

	if !params.ExactIn && outputAmount.Lt(amountSpecified) {
		return nil, errors.New("BunniHook__InsufficientOutput")
	}

	if (params.ZeroForOne && updatedSqrtPriceX96.Gt(h.Slot0.SqrtPriceX96)) ||
		(!params.ZeroForOne && updatedSqrtPriceX96.Lt(h.Slot0.SqrtPriceX96)) ||
		(outputAmount.IsZero() || inputAmount.IsZero()) {

		return nil, errors.New("BunniHook__InvalidSwap")
	}

	lastSurgeTimestamp := h.Slot0.LastSurgeTimestamp

	if shouldSurge {
		timeSinceLastSwap := blockTimestamp - h.Slot0.LastSwapTimestamp

		surgeFeeAutostartThreshold := uint32(h.HookParams.SurgeFeeAutostartThreshold)
		if timeSinceLastSwap >= surgeFeeAutostartThreshold {
			lastSurgeTimestamp = h.Slot0.LastSwapTimestamp + surgeFeeAutostartThreshold
		} else {
			lastSurgeTimestamp = blockTimestamp
		}
	}

	h.Slot0.SqrtPriceX96 = updatedSqrtPriceX96
	h.Slot0.Tick = updatedTick
	h.Slot0.LastSwapTimestamp = blockTimestamp
	h.Slot0.LastSurgeTimestamp = lastSurgeTimestamp

	var amAmmSwapFee uint256.Int
	if h.HookParams.AmAmmEnabled {
		if params.ZeroForOne {
			amAmmSwapFee.Set(h.AmAmm.SwapFee0For1)
		} else {
			amAmmSwapFee.Set(h.AmAmm.SwapFee1For0)
		}
	}

	useAmAmmFee := h.HookParams.AmAmmEnabled && !valueobject.IsZeroAddress(h.AmAmm.AmAmmManager)

	var hookFeesBaseSwapFee uint256.Int
	if feeOverridden {
		surgeFee, err := computeSurgeFee(blockTimestamp, lastSurgeTimestamp, h.HookParams.SurgeFeeHalfLife)
		if err != nil {
			return nil, err
		}

		hookFeesBaseSwapFee.Set(u256.Max(feeOverride, surgeFee))
	} else {
		dynamicFee, err := computeDynamicSwapFee(blockTimestamp, updatedSqrtPriceX96, int(feeMeanTick), lastSurgeTimestamp,
			h.HookParams.FeeMin, h.HookParams.FeeMax, h.HookParams.FeeQuadraticMultiplier, h.HookParams.SurgeFeeHalfLife)
		if err != nil {
			return nil, err
		}

		hookFeesBaseSwapFee.Set(dynamicFee)
	}

	var swapFee uint256.Int
	var swapFeeAmount *uint256.Int
	var hookFeesAmount *uint256.Int
	var curatorFeeAmount *uint256.Int
	var hookHandleSwapInputAmount uint256.Int
	var hookHandleSwapOutputAmount uint256.Int

	var result uniswapv4.BeforeSwapHookResult

	if useAmAmmFee {
		surgeFee, err := computeSurgeFee(blockTimestamp, lastSurgeTimestamp, h.HookParams.SurgeFeeHalfLife)
		if err != nil {
			return nil, err
		}

		swapFee.Set(u256.Max(&amAmmSwapFee, surgeFee))
	} else {
		swapFee.Set(&hookFeesBaseSwapFee)
	}

	if params.ExactIn {
		swapFeeAmount, err = v3Utils.MulDivRoundingUp(outputAmount, &swapFee, SWAP_FEE_BASE)
		if err != nil {
			return nil, err
		}

		if useAmAmmFee {
			baseSwapFeeAmount, err := v3Utils.MulDivRoundingUp(outputAmount, &hookFeesBaseSwapFee, SWAP_FEE_BASE)
			if err != nil {
				return nil, err
			}

			hookFeesAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.HookFee, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.CuratorFees.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			if swapFee.Cmp(&amAmmSwapFee) != 0 {
				swapFeeAdjusted, err := v3Utils.MulDivRoundingUp(&hookFeesBaseSwapFee, h.HookFee, MODIFIER_BASE)
				if err != nil {
					return nil, err
				}

				swapFeeAdjusted.Sub(&swapFee, swapFeeAdjusted)

				tmp, err := v3Utils.MulDivRoundingUp(&hookFeesBaseSwapFee, h.CuratorFees.FeeRate, CURATOR_FEE_BASE)
				if err != nil {
					return nil, err
				}

				swapFeeAdjusted.Sub(swapFeeAdjusted, tmp)

				if swapFeeAdjusted.Lt(&amAmmSwapFee) {
					swapFeeAdjusted.Set(&amAmmSwapFee)
				}

				swapFeeAmount, err = v3Utils.MulDivRoundingUp(outputAmount, swapFeeAdjusted, SWAP_FEE_BASE)
				if err != nil {
					return nil, err
				}
			}
		} else {
			hookFeesAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.HookFee, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.CuratorFees.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount)
			swapFeeAmount.Sub(swapFeeAmount, curatorFeeAmount)
		}

		outputAmount.Sub(outputAmount, swapFeeAmount)
		outputAmount.Sub(outputAmount, hookFeesAmount)
		outputAmount.Sub(outputAmount, curatorFeeAmount)

		actualInputAmount := u256.Max(amountSpecified, inputAmount)
		result.DeltaSpecific = actualInputAmount.ToBig()

		outputAmountInt := outputAmount.ToBig()
		result.DeltaUnSpecific = outputAmountInt.Neg(outputAmountInt)

		hookHandleSwapInputAmount.Set(inputAmount)

		hookHandleSwapOutputAmount.Add(outputAmount, hookFeesAmount)
		if useAmAmmFee {
			hookHandleSwapOutputAmount.Add(&hookHandleSwapOutputAmount, swapFeeAmount)
		}

	} else {
		swapFeeAmount, err = v3Utils.MulDivRoundingUp(inputAmount, &swapFee, new(uint256.Int).Sub(SWAP_FEE_BASE, &swapFee))
		if err != nil {
			return nil, err
		}

		if useAmAmmFee {
			baseSwapFeeAmount, err := v3Utils.MulDivRoundingUp(inputAmount, &hookFeesBaseSwapFee,
				new(uint256.Int).Sub(SWAP_FEE_BASE, &hookFeesBaseSwapFee))
			if err != nil {
				return nil, err
			}

			hookFeesAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.HookFee, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.CuratorFees.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}
		} else {
			hookFeesAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.HookFee, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.CuratorFees.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount)
			swapFeeAmount.Sub(swapFeeAmount, curatorFeeAmount)
		}

		inputAmount.Add(inputAmount, swapFeeAmount)
		inputAmount.Add(inputAmount, hookFeesAmount)
		inputAmount.Add(inputAmount, curatorFeeAmount)

		result.DeltaUnSpecific = inputAmount.ToBig()

		actualOutputAmount := u256.Min(amountSpecified, outputAmount).ToBig()
		result.DeltaSpecific = actualOutputAmount.Neg(actualOutputAmount)

		hookHandleSwapOutputAmount.Set(outputAmount)

		hookHandleSwapInputAmount.Sub(inputAmount, hookFeesAmount)
		if useAmAmmFee {
			hookHandleSwapInputAmount.Sub(&hookHandleSwapInputAmount, swapFeeAmount)
		}
	}

	err = h.hookHandleSwap(params.ZeroForOne, &hookHandleSwapInputAmount, &hookHandleSwapOutputAmount, shouldSurge)
	if err != nil {
		return nil, err
	}

	// rebalanceOrderDeadline := h.rebalanceOrderDeadline
	// if shouldSurge {
	// 	rebalanceOrderDeadline = uint256.NewInt(0)
	// }

	// if h.HookParams.RebalanceThreshold != 0 &&
	// 	(shouldSurge || blockTimestamp > uint32(rebalanceOrderDeadline.Uint64()) && !rebalanceOrderDeadline.IsZero()) {
	// 	if shouldSurge {
	// 		h.rebalanceOrderDeadline.Clear()
	// 	}

	// 	h.rebalance()
	// }

	h.hooklet.AfterSwap(nil)

	return &result, nil
}

func (h *Hook) hookHandleSwap(
	zeroForOne bool,
	inputAmount,
	outputAmount *uint256.Int,
	shouldSurge bool,
) error {
	if !inputAmount.IsZero() {
		if zeroForOne {
			h.BunniState.RawBalance0.Add(h.BunniState.RawBalance0, inputAmount)
		} else {
			h.BunniState.RawBalance1.Add(h.BunniState.RawBalance1, inputAmount)
		}
	}

	if !outputAmount.IsZero() {
		outputRawBalance, vaultIndex := h.BunniState.RawBalance0, 0
		if zeroForOne {
			outputRawBalance, vaultIndex = h.BunniState.RawBalance1, 1
		}

		outputVault := h.Vaults[vaultIndex]

		if !valueobject.IsZeroAddress(outputVault.Address) && outputRawBalance.Lt(outputAmount) {
			rawBalanceChange := i256.SafeToInt256(new(uint256.Int).Sub(outputAmount, outputRawBalance))

			reserveChange, rawBalanceChange, err := h.updateVaultReserveViaClaimTokens(vaultIndex, rawBalanceChange)
			if err != nil {
				return err
			}

			if zeroForOne {
				h.BunniState.Reserve1.Add(h.BunniState.Reserve1, i256.SafeConvertToUInt256(reserveChange))
				h.BunniState.RawBalance1.Add(h.BunniState.RawBalance1, i256.SafeConvertToUInt256(rawBalanceChange))
			} else {
				h.BunniState.Reserve0.Add(h.BunniState.Reserve0, i256.SafeConvertToUInt256(reserveChange))
				h.BunniState.RawBalance0.Add(h.BunniState.RawBalance0, i256.SafeConvertToUInt256(rawBalanceChange))
			}
		}

		if zeroForOne {
			h.BunniState.RawBalance1.Sub(h.BunniState.RawBalance1, outputAmount)
		} else {
			h.BunniState.RawBalance0.Sub(h.BunniState.RawBalance0, outputAmount)
		}
	}

	if !shouldSurge {
		if !valueobject.IsZeroAddress(h.Vaults[0].Address) {
			newReserve0, newRawBalance0, err := h.updateRawBalanceIfNeeded(
				0,
				h.BunniState.RawBalance0, h.BunniState.Reserve0,
				h.BunniState.MinRawTokenRatio0, h.BunniState.MaxRawTokenRatio0, h.BunniState.TargetRawTokenRatio0)
			if err != nil {
				return err
			}

			h.BunniState.Reserve0.Set(newReserve0)
			h.BunniState.RawBalance0.Set(newRawBalance0)
		}

		if !valueobject.IsZeroAddress(h.Vaults[1].Address) {
			newReserve1, newRawBalance1, err := h.updateRawBalanceIfNeeded(
				1,
				h.BunniState.RawBalance1, h.BunniState.Reserve1,
				h.BunniState.MinRawTokenRatio1, h.BunniState.MaxRawTokenRatio1, h.BunniState.TargetRawTokenRatio1)
			if err != nil {
				return err
			}

			h.BunniState.Reserve1.Set(newReserve1)
			h.BunniState.RawBalance1.Set(newRawBalance1)
		}
	}

	return nil
}

func (h *Hook) updateRawBalanceIfNeeded(
	vaultIndex int,
	rawBalance,
	reserve,
	minRatio,
	maxRatio,
	targetRatio *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	reserveInUnderlying := getReservesInUnderlying(h.Vaults[vaultIndex], reserve)

	var balance uint256.Int
	balance.Add(rawBalance, reserveInUnderlying)

	var maxRawBalance uint256.Int
	maxRawBalance.MulDivOverflow(&balance, maxRatio, RAW_TOKEN_RATIO_BASE)

	minRawBalance := reserveInUnderlying // reuse
	minRawBalance.MulDivOverflow(&balance, minRatio, RAW_TOKEN_RATIO_BASE)

	if rawBalance.Lt(minRawBalance) || rawBalance.Gt(&maxRawBalance) {
		targetRawBalance := maxRawBalance // reuse
		targetRawBalance.MulDivOverflow(&balance, targetRatio, RAW_TOKEN_RATIO_BASE)

		rawBalanceChange := i256.SafeToInt256(&targetRawBalance)
		rawBalanceChange.Sub(rawBalanceChange, i256.SafeToInt256(rawBalance))

		reserveChange, rawBalanceChange, err := h.updateVaultReserveViaClaimTokens(vaultIndex, rawBalanceChange)
		if err != nil {
			return nil, nil, err
		}

		newReserveInt := i256.SafeToInt256(reserve)
		newRawBalanceInt := i256.SafeToInt256(rawBalance)

		return i256.SafeConvertToUInt256(newReserveInt.Add(newReserveInt, reserveChange)),
			i256.SafeConvertToUInt256(newRawBalanceInt.Add(newRawBalanceInt, rawBalanceChange)),
			nil
	}

	return reserve, rawBalance, nil
}

func getReservesInUnderlying(vault Vault, reserveAmount *uint256.Int) *uint256.Int {
	if valueobject.IsZeroAddress(vault.Address) {
		return reserveAmount
	}

	reserve, _ := v3Utils.MulDivRoundingUp(reserveAmount, WAD, vault.RedeemRate)
	return reserve
}

func (h *Hook) updateVaultReserveViaClaimTokens(
	index int,
	rawBalanceChange *int256.Int,
) (*int256.Int, *int256.Int, error) {
	absAmount := math.Abs(rawBalanceChange)

	var (
		reserveChange          int256.Int
		actualRawBalanceChange int256.Int
	)

	if rawBalanceChange.Sign() < 0 {
		absAmount = u256.Min(u256.Min(absAmount, h.Vaults[index].MaxDeposit), h.PoolManagerReserves[index])

		if absAmount.IsZero() {
			return int256.NewInt(0), int256.NewInt(0), nil
		}

		depositAmt := absAmount

		reserveChange.Set(i256.SafeToInt256(depositAmt))

		// expect deposited amount to be equal to absAmount
		actualDepositedAmount := absAmount

		actualRawBalanceChange.Set(i256.SafeToInt256(actualDepositedAmount))
		actualRawBalanceChange.Neg(&actualRawBalanceChange)

	} else if rawBalanceChange.Sign() > 0 {
		withdrawAmt := absAmount

		withdrawAmtInt := i256.SafeToInt256(withdrawAmt)
		withdrawAmtInt.Neg(withdrawAmtInt)

		reserveChange.Set(withdrawAmtInt)

		if h.isNative[index] {
			actualRawBalanceChange.Set(rawBalanceChange)
		}
	}

	return &reserveChange, &actualRawBalanceChange, nil
}

type BunniComputeSwapInput struct {
	TotalLiquidity                   *uint256.Int
	LiquidityDensityOfRoundedTickX96 *uint256.Int
	CurrentActiveBalance0            *uint256.Int
	CurrentActiveBalance1            *uint256.Int
	ArithmeticMeanTick               int
	ZeroForOne                       bool
	ExactIn                          bool
	AmountSpecified                  *uint256.Int
	SqrtPriceLimitX96                *uint256.Int
	LdfState                         [32]byte
}

func (h *Hook) computeSwap(input BunniComputeSwapInput) (*uint256.Int, int, *uint256.Int, *uint256.Int, error) {
	var inputAmount, outputAmount uint256.Int
	if input.ExactIn {
		inputAmount.Set(input.AmountSpecified)
	} else {
		outputAmount.Set(input.AmountSpecified)
	}

	updatedTick := h.Slot0.Tick

	var updatedRoundedTickLiquidity uint256.Int
	updatedRoundedTickLiquidity.Mul(input.TotalLiquidity, input.LiquidityDensityOfRoundedTickX96)
	updatedRoundedTickLiquidity.Rsh(&updatedRoundedTickLiquidity, 96)

	var sqrtPriceLimitX96 uint256.Int
	sqrtPriceLimitX96.Set(input.SqrtPriceLimitX96)

	minSqrtPrice, err := math.GetSqrtPriceAtTick(math.MinUsableTick(h.tickSpacing))
	if err != nil {
		return nil, 0, nil, nil, err
	}
	maxSqrtPrice, err := math.GetSqrtPriceAtTick(math.MaxUsableTick(h.tickSpacing))
	if err != nil {
		return nil, 0, nil, nil, err
	}

	if (input.ZeroForOne && sqrtPriceLimitX96.Cmp(minSqrtPrice) <= 0) ||
		(!input.ZeroForOne && sqrtPriceLimitX96.Cmp(maxSqrtPrice) >= 0) {
		if input.ZeroForOne {
			sqrtPriceLimitX96.AddUint64(minSqrtPrice, 1)
		} else {
			sqrtPriceLimitX96.SubUint64(maxSqrtPrice, 1)
		}
	}

	roundedTick, nextRoundedTick := math.RoundTick(h.Slot0.Tick, h.tickSpacing)

	var naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut *uint256.Int

	if !updatedRoundedTickLiquidity.IsZero() {
		tickNext := lo.Ternary(input.ZeroForOne, roundedTick, nextRoundedTick)

		sqrtPriceNextX96, err := math.GetSqrtPriceAtTick(tickNext)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		var sqrtPriceTargetX96 uint256.Int
		if input.ZeroForOne {
			if sqrtPriceLimitX96.Cmp(sqrtPriceNextX96) > 0 {
				sqrtPriceTargetX96.Set(&sqrtPriceLimitX96)
			} else {
				sqrtPriceTargetX96.Set(sqrtPriceNextX96)
			}
		} else {
			if sqrtPriceLimitX96.Cmp(sqrtPriceNextX96) < 0 {
				sqrtPriceTargetX96.Set(&sqrtPriceLimitX96)
			} else {
				sqrtPriceTargetX96.Set(sqrtPriceNextX96)
			}
		}

		naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut, err := math.ComputeSwapStep(
			input.ExactIn, input.ZeroForOne, h.Slot0.SqrtPriceX96,
			&sqrtPriceTargetX96, &updatedRoundedTickLiquidity, input.AmountSpecified, u256.U0)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		if (input.ExactIn && naiveSwapAmountIn.Eq(input.AmountSpecified)) ||
			(!input.ExactIn && naiveSwapAmountOut.Eq(input.AmountSpecified)) {

			if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceNextX96) {
				updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
			} else if !naiveSwapResultSqrtPriceX96.Eq(h.Slot0.SqrtPriceX96) {
				updatedTick, err = math.GetTickAtSqrtPrice(naiveSwapResultSqrtPriceX96)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}

			currentBalance := lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0)
			if naiveSwapAmountOut.Gt(currentBalance) {
				naiveSwapAmountOut.Set(currentBalance)
			}

			return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
		}
	}

	var inverseCumulativeAmountFnInput uint256.Int
	if input.ExactIn {
		inverseCumulativeAmountFnInput.Set(lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance0, input.CurrentActiveBalance1))
		inverseCumulativeAmountFnInput.Add(&inverseCumulativeAmountFnInput, &inputAmount)
	} else {
		inverseCumulativeAmountFnInput.Set(lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0))
		inverseCumulativeAmountFnInput.Sub(&inverseCumulativeAmountFnInput, &outputAmount)
	}
	success, updatedRoundedTick, cumulativeAmount0, cumulativeAmount1, swapLiquidity, err := h.ldf.ComputeSwap(
		&inverseCumulativeAmountFnInput,
		input.TotalLiquidity,
		input.ZeroForOne,
		input.ExactIn,
		input.ArithmeticMeanTick,
		h.Slot0.Tick,
		h.BunniState.LdfParams,
		h.LdfState,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	if success {
		if (input.ZeroForOne && updatedRoundedTick >= roundedTick) ||
			(!input.ZeroForOne && updatedRoundedTick <= roundedTick) {

			if updatedRoundedTickLiquidity.IsZero() {
				return h.Slot0.SqrtPriceX96, h.Slot0.Tick, uint256.NewInt(0), uint256.NewInt(0), nil
			}

			tickNext := lo.Ternary(input.ZeroForOne, roundedTick, nextRoundedTick)

			sqrtPriceNextX96, err := math.GetSqrtPriceAtTick(tickNext)
			if err != nil {
				return nil, 0, nil, nil, err
			}

			if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceNextX96) {
				updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
			} else if !naiveSwapResultSqrtPriceX96.Eq(h.Slot0.SqrtPriceX96) {
				updatedTick, err = math.GetTickAtSqrtPrice(naiveSwapResultSqrtPriceX96)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}

			currentBalance := lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0)
			if naiveSwapAmountOut.Gt(currentBalance) {
				naiveSwapAmountOut.Set(currentBalance)
			}

			return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
		}

		tickStart := updatedRoundedTick
		tickNext := updatedRoundedTick + h.tickSpacing
		if !input.ZeroForOne {
			tickStart, tickNext = tickNext, tickStart
		}

		startSqrtPriceX96, err := math.GetSqrtPriceAtTick(tickStart)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		if input.ZeroForOne && sqrtPriceLimitX96.Lt(startSqrtPriceX96) ||
			(!input.ZeroForOne && sqrtPriceLimitX96.Gt(startSqrtPriceX96)) {

			var sqrtPriceNextX96 uint256.Int
			err = v3Utils.GetSqrtRatioAtTickV2(tickNext, &sqrtPriceNextX96)
			if err != nil {
				return nil, 0, nil, nil, err
			}

			if input.ZeroForOne {
				if cumulativeAmount0.Lt(input.CurrentActiveBalance0) {
					cumulativeAmount0.Set(input.CurrentActiveBalance0)
				}

				if cumulativeAmount1.Lt(input.CurrentActiveBalance1) {
					cumulativeAmount1.Set(input.CurrentActiveBalance1)
				}

				var hitSqrtPriceLimit bool
				if swapLiquidity.IsZero() || sqrtPriceLimitX96.Eq(startSqrtPriceX96) {
					naiveSwapResultSqrtPriceX96.Set(startSqrtPriceX96)
					naiveSwapAmountIn.Clear()
					naiveSwapAmountOut.Clear()
				} else {
					var amountSpecifiedRemaining int256.Int
					if input.ExactIn {
						var temp uint256.Int
						temp.Sub(&inverseCumulativeAmountFnInput, cumulativeAmount0)
						amountSpecifiedRemaining.Set(i256.SafeToInt256(&temp))
						amountSpecifiedRemaining.Neg(&amountSpecifiedRemaining)
					} else {
						var temp uint256.Int
						temp.Sub(cumulativeAmount1, &inverseCumulativeAmountFnInput)
						amountSpecifiedRemaining.Set(i256.SafeToInt256(&temp))
					}

					naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut, err = math.ComputeSwapStep(
						input.ExactIn, input.ZeroForOne, startSqrtPriceX96, &sqrtPriceNextX96, swapLiquidity,
						i256.SafeConvertToUInt256(&amountSpecifiedRemaining), EPSILON_FEE)
					if err != nil {
						return nil, 0, nil, nil, err
					}

					if naiveSwapResultSqrtPriceX96.Eq(&sqrtPriceLimitX96) && !sqrtPriceLimitX96.Eq(&sqrtPriceNextX96) {
						hitSqrtPriceLimit = true
					}
				}

				if !hitSqrtPriceLimit {
					updatedTick = tickStart

					if naiveSwapResultSqrtPriceX96.Eq(&sqrtPriceNextX96) {
						updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
					} else if !naiveSwapResultSqrtPriceX96.Eq(startSqrtPriceX96) {
						updatedTick, err = math.GetTickAtSqrtPrice(naiveSwapResultSqrtPriceX96)
						if err != nil {
							return nil, 0, nil, nil, err
						}
					}

					var updatedSqrtPriceX96 uint256.Int
					updatedSqrtPriceX96.Set(naiveSwapResultSqrtPriceX96)

					if (input.ExactIn && naiveSwapAmountIn.Eq(input.AmountSpecified)) ||
						(!input.ExactIn && naiveSwapAmountOut.Eq(input.AmountSpecified)) {

						currentBalance := lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0)
						if naiveSwapAmountOut.Gt(currentBalance) {
							naiveSwapAmountOut.Set(currentBalance)
						}

						return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
					}

					if (input.ZeroForOne && cumulativeAmount1.Lt(naiveSwapAmountOut)) ||
						(!input.ZeroForOne && cumulativeAmount0.Lt(naiveSwapAmountOut)) {
						return nil, 0, nil, nil, errors.New("BunniSwapMath__SwapFailed")
					}

					var updatedActiveBalance0, updatedActiveBalance1 uint256.Int
					if input.ZeroForOne {
						updatedActiveBalance0.Add(cumulativeAmount0, naiveSwapAmountIn)
						updatedActiveBalance1.Sub(cumulativeAmount1, naiveSwapAmountOut)
					} else {
						updatedActiveBalance0.Sub(cumulativeAmount0, naiveSwapAmountOut)
						updatedActiveBalance1.Add(cumulativeAmount1, naiveSwapAmountIn)
					}

					var finalInputAmount, finalOutputAmount uint256.Int
					if input.ZeroForOne {
						finalInputAmount.Sub(&updatedActiveBalance0, input.CurrentActiveBalance0)
						finalOutputAmount.Set(math.SubReLU(input.CurrentActiveBalance1, &updatedActiveBalance1))
					} else {
						finalInputAmount.Sub(&updatedActiveBalance1, input.CurrentActiveBalance1)
						finalOutputAmount.Set(math.SubReLU(input.CurrentActiveBalance0, &updatedActiveBalance0))
					}

					return &updatedSqrtPriceX96, updatedTick, &finalInputAmount, &finalOutputAmount, nil
				}
			}
		}
	}

	var updatedSqrtPriceX96 uint256.Int
	updatedSqrtPriceX96.Set(&sqrtPriceLimitX96)

	if sqrtPriceLimitX96.Eq(h.Slot0.SqrtPriceX96) {
		updatedTick = h.Slot0.Tick
	} else {
		updatedTick, err = math.GetTickAtSqrtPrice(&sqrtPriceLimitX96)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}

	totalDensity0X96, totalDensity1X96, _, _, _, _, err := h.queryLDF(&updatedSqrtPriceX96, updatedTick,
		input.ArithmeticMeanTick, input.LdfState, u256.U0, u256.U0, ZERO_BALANCE)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	var updatedActiveBalance0, updatedActiveBalance1 uint256.Int
	updatedActiveBalance0.MulDivOverflow(totalDensity0X96, input.TotalLiquidity, SWAP_FEE_BASE)
	updatedActiveBalance1.MulDivOverflow(totalDensity1X96, input.TotalLiquidity, SWAP_FEE_BASE)

	var finalInputAmount, finalOutputAmount uint256.Int
	if input.ZeroForOne {
		finalInputAmount.Sub(&updatedActiveBalance0, input.CurrentActiveBalance0)
		finalOutputAmount.Set(math.SubReLU(input.CurrentActiveBalance1, &updatedActiveBalance1))
	} else {
		finalInputAmount.Sub(&updatedActiveBalance1, input.CurrentActiveBalance1)
		finalOutputAmount.Set(math.SubReLU(input.CurrentActiveBalance0, &updatedActiveBalance0))
	}

	return &updatedSqrtPriceX96, updatedTick, &finalInputAmount, &finalOutputAmount, nil
}

func (h *Hook) queryLDF(
	sqrtPriceX96 *uint256.Int,
	tick,
	arithmeticMeanTick int,
	ldfState [32]byte,
	balance0,
	balance1 *uint256.Int,
	idleBalance [32]byte,
) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, [32]byte, bool, error) {

	roundedTick, nextRoundedTick := math.RoundTick(h.Slot0.Tick, h.tickSpacing)

	roundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}
	nextRoundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(nextRoundedTick)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}

	liquidityDensityOfRoundedTickX96, density0RightOfRoundedTickX96, density1LeftOfRoundedTickX96,
		newLdfState, shouldSurge, err := h.ldf.Query(
		roundedTick,
		arithmeticMeanTick,
		tick,
		h.BunniState.LdfParams,
		h.LdfState,
	)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}

	var density0OfRoundedTickX96, density1OfRoundedTickX96 *uint256.Int
	density0OfRoundedTickX96, density1OfRoundedTickX96, err = getAmountsForLiquidity(
		sqrtPriceX96,
		roundedTickSqrtRatio,
		nextRoundedTickSqrtRatio,
		liquidityDensityOfRoundedTickX96,
		true,
	)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}

	var totalDensity0X96, totalDensity1X96 uint256.Int
	totalDensity0X96.Add(density0RightOfRoundedTickX96, density0OfRoundedTickX96)
	totalDensity1X96.Add(density1LeftOfRoundedTickX96, density1OfRoundedTickX96)

	var modifiedBalance0, modifiedBalance1 uint256.Int
	modifiedBalance0.Set(balance0)
	modifiedBalance1.Set(balance1)

	if !shouldSurge {
		idleBalance, isToken0 := math.FromIdleBalance(idleBalance)
		if isToken0 {
			modifiedBalance0.Set(math.SubReLU(&modifiedBalance0, idleBalance))
		} else {
			modifiedBalance1.Set(math.SubReLU(&modifiedBalance1, idleBalance))
		}
	}

	var totalLiquidity uint256.Int
	var activeBalance0, activeBalance1 uint256.Int

	if !modifiedBalance0.IsZero() || !modifiedBalance1.IsZero() {
		noToken0 := modifiedBalance0.IsZero() || totalDensity0X96.IsZero()
		noToken1 := modifiedBalance1.IsZero() || totalDensity1X96.IsZero()

		var totalLiquidityEstimate0, totalLiquidityEstimate1 *uint256.Int
		if !noToken0 {
			totalLiquidityEstimate0, err = math.FullMulDiv(&modifiedBalance0, Q96, &totalDensity0X96)
			if err != nil {
				return nil, nil, nil, nil, ldfState, false, err
			}
		}
		if !noToken1 {
			totalLiquidityEstimate1, err = math.FullMulDiv(&modifiedBalance1, Q96, &totalDensity1X96)
			if err != nil {
				return nil, nil, nil, nil, ldfState, false, err
			}
		}

		useLiquidityEstimate0 := (totalLiquidityEstimate0.Lt(totalLiquidityEstimate1) ||
			totalDensity1X96.IsZero()) && !totalDensity0X96.IsZero()

		if useLiquidityEstimate0 {
			if !noToken0 {
				totalLiquidityPtr, err := math.RoundUpFullMulDivResult(&modifiedBalance0, Q96, &totalDensity0X96, totalLiquidityEstimate0)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				totalLiquidity.Set(totalLiquidityPtr)
			}

			var temp0, temp1 *uint256.Int
			if !noToken0 {
				temp0, err = math.FullMulX96(totalLiquidityEstimate0, &totalDensity0X96)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance0.Set(u256.Min(&modifiedBalance0, temp0))
			}
			if !noToken1 {
				temp1, err = math.FullMulX96(totalLiquidityEstimate0, &totalDensity1X96)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance1.Set(u256.Min(&modifiedBalance1, temp1))
			}
		} else {
			if !noToken1 {
				totalLiquidityPtr, err := math.RoundUpFullMulDivResult(&modifiedBalance1, Q96, &totalDensity1X96, totalLiquidityEstimate1)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				totalLiquidity.Set(totalLiquidityPtr)
			}

			var temp0, temp1 *uint256.Int
			if !noToken0 {
				temp0, err = math.FullMulX96(totalLiquidityEstimate1, &totalDensity0X96)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance0.Set(u256.Min(&modifiedBalance0, temp0))
			}
			if !noToken1 {
				temp1, err = math.FullMulX96(totalLiquidityEstimate1, &totalDensity1X96)
				if err != nil {
					return nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance1.Set(u256.Min(&modifiedBalance1, temp1))
			}
		}
	}

	return &totalLiquidity, &totalDensity0X96, &totalDensity1X96, liquidityDensityOfRoundedTickX96,
		newLdfState, shouldSurge, nil
}

func getAmountsForLiquidity(
	sqrtPriceX96,
	sqrtPriceAX96,
	sqrtPriceBX96,
	liquidity *uint256.Int,
	roundUp bool,
) (*uint256.Int, *uint256.Int, error) {
	var amount0, amount1 uint256.Int

	if sqrtPriceAX96.Gt(sqrtPriceBX96) {
		sqrtPriceAX96, sqrtPriceBX96 = sqrtPriceBX96, sqrtPriceAX96
	}

	if sqrtPriceX96.Cmp(sqrtPriceAX96) <= 0 {
		amount0Delta, err := math.GetAmount0Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount0.Set(amount0Delta)
	} else if sqrtPriceX96.Lt(sqrtPriceBX96) {
		amount0Delta, err := math.GetAmount0Delta(sqrtPriceX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount0.Set(amount0Delta)

		amount1Delta, err := math.GetAmount1Delta(sqrtPriceAX96, sqrtPriceX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount1.Set(amount1Delta)
	} else {
		amount1Delta, err := math.GetAmount1Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount1.Set(amount1Delta)
	}

	return &amount0, &amount1, nil
}

func (h *Hook) getTwap(blockTimestamp uint32) (int64, error) {
	tickCumulatives, err := h.oracle.ObserveDouble(
		h.ObservationState.IntermediateObservation,
		blockTimestamp,
		[]uint32{h.BunniState.TwapSecondsAgo, 0},
		h.Slot0.Tick,
		h.ObservationState.Index,
		h.ObservationState.Cardinality,
	)
	if err != nil {
		return 0, err
	}

	tickCumulativesDelta := tickCumulatives[0] - tickCumulatives[1]

	return tickCumulativesDelta / int64(h.BunniState.TwapSecondsAgo), nil
}

func (h *Hook) updateOracle(blockTimestamp uint32) {
	h.ObservationState.IntermediateObservation, h.ObservationState.Index, h.ObservationState.Cardinality =
		h.oracle.Write(h.ObservationState.IntermediateObservation,
			h.ObservationState.Index, blockTimestamp, h.Slot0.Tick, h.ObservationState.Cardinality,
			h.ObservationState.CardinalityNext, h.HookParams.OracleMinInterval)
}

func computeSurgeFee(
	blockTimestamp,
	lastSurgeTimestamp uint32,
	surgeFeeHalfLife *uint256.Int,
) (fee *uint256.Int, err error) {
	timeSinceLastSurge := uint256.NewInt(uint64(blockTimestamp - lastSurgeTimestamp))

	fee, _ = new(uint256.Int).MulDivOverflow(timeSinceLastSurge, LN2_WAD, surgeFeeHalfLife)

	feeInt := i256.SafeToInt256(fee)
	feeInt.Neg(feeInt)

	fee, err = math.ExpWad(feeInt)
	if err != nil {
		return nil, err
	}

	fee, err = math.MulWadUp(SWAP_FEE_BASE, fee)
	if err != nil {
		return nil, err
	}

	return
}

func computeDynamicSwapFee(
	blockTimestamp uint32,
	postSwapSqrtPriceX96 *uint256.Int,
	arithmeticMeanTick int,
	lastSurgeTimestamp uint32,
	feeMin,
	feeMax,
	feeQuadraticMultiplier,
	surgeFeeHalfLife *uint256.Int,
) (*uint256.Int, error) {

	fee, err := computeSurgeFee(blockTimestamp, lastSurgeTimestamp, surgeFeeHalfLife)
	if err != nil {
		return nil, err
	}

	if feeQuadraticMultiplier.IsZero() || feeMin.Eq(feeMax) {
		return u256.Max(feeMin, fee), nil
	}

	sqrtPriceX96, err := math.GetSqrtPriceAtTick(arithmeticMeanTick)
	if err != nil {
		return nil, err
	}

	var ratio uint256.Int
	ratio.MulDivOverflow(postSwapSqrtPriceX96, SWAP_FEE_BASE, sqrtPriceX96)

	if ratio.Gt(MAX_SWAP_FEE_RATIO) {
		ratio.Set(MAX_SWAP_FEE_RATIO)
	}

	ratio.MulDivOverflow(&ratio, &ratio, SWAP_FEE_BASE)
	delta := math.Dist(&ratio, SWAP_FEE_BASE)

	delta.Exp(delta, u256.U2)

	quadraticTerm := sqrtPriceX96
	err = v3Utils.MulDivRoundingUpV2(feeQuadraticMultiplier, delta, SWAP_FEE_BASE_SQUARED, quadraticTerm)
	if err != nil {
		return nil, err
	}

	return u256.Max(fee, u256.Min(quadraticTerm.Add(quadraticTerm, feeMin), feeMax)), nil
}

func (h *Hook) shouldSurgeFromVaults() (shouldSurge bool, err error) {
	if !valueobject.IsZeroAddress(h.Vaults[0].Address) || !valueobject.IsZeroAddress(h.Vaults[1].Address) {
		rescaleFactor0 := 18 + h.Vaults[0].Decimals - h.BunniState.Currency0Decimals
		rescaleFactor1 := 18 + h.Vaults[1].Decimals - h.BunniState.Currency1Decimals

		var sharePrice0 *uint256.Int
		if !h.BunniState.Reserve0.IsZero() {
			reserveBalance0 := getReservesInUnderlying(h.Vaults[0], h.BunniState.Reserve0)
			sharePrice0, err = v3Utils.MulDivRoundingUp(reserveBalance0, u256.TenPow(rescaleFactor0), h.BunniState.Reserve0)
			if err != nil {
				return false, err
			}
		}

		var sharePrice1 *uint256.Int
		if !h.BunniState.Reserve1.IsZero() {
			reserveBalance1 := getReservesInUnderlying(h.Vaults[1], h.BunniState.Reserve1)
			sharePrice1, err = v3Utils.MulDivRoundingUp(reserveBalance1, u256.TenPow(rescaleFactor1), h.BunniState.Reserve1)
			if err != nil {
				return false, err
			}
		}

		shouldSurge = h.VaultSharePrices.Initialized &&
			(math.Dist(sharePrice0, h.VaultSharePrices.SharedPrice0).
				Gt(new(uint256.Int).Div(h.VaultSharePrices.SharedPrice0, h.HookParams.VaultSurgeThreshold0)) ||
				math.Dist(sharePrice1, h.VaultSharePrices.SharedPrice1).
					Gt(new(uint256.Int).Div(h.VaultSharePrices.SharedPrice1, h.HookParams.VaultSurgeThreshold1)))

		if !h.VaultSharePrices.Initialized || !sharePrice0.Eq(h.VaultSharePrices.SharedPrice0) ||
			!sharePrice1.Eq(h.VaultSharePrices.SharedPrice1) {
			h.VaultSharePrices.Initialized = true
			h.VaultSharePrices.SharedPrice0.Set(sharePrice0)
			h.VaultSharePrices.SharedPrice1.Set(sharePrice1)
		}
	}

	return shouldSurge, nil
}

func (h *Hook) computeIdleBalance(activeBalance0, activeBalance1, balance0, balance1 *uint256.Int) ([32]byte, error) {
	extraBalance0 := math.SubReLU(balance0, activeBalance0)
	extraBalance1 := math.SubReLU(balance1, activeBalance1)

	var extraBalanceProportion0, extraBalanceProportion1 uint256.Int
	if !balance0.IsZero() {
		extraBalanceProportion0.Mul(extraBalance0, math.WAD)
		extraBalanceProportion0.Div(&extraBalanceProportion0, balance0)
	}

	if !balance1.IsZero() {
		extraBalanceProportion1.Mul(extraBalance1, math.WAD)
		extraBalanceProportion1.Div(&extraBalanceProportion1, balance1)
	}

	isToken0 := extraBalanceProportion0.Cmp(&extraBalanceProportion1) >= 0

	var idleBalance [32]byte
	var err error
	if isToken0 {
		idleBalance, err = math.ToIdleBalance(extraBalance0, true)
	} else {
		idleBalance, err = math.ToIdleBalance(extraBalance1, false)
	}
	if err != nil {
		return [32]byte{}, err
	}

	return idleBalance, nil
}
