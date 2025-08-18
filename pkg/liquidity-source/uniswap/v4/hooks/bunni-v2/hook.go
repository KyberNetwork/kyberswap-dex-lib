package bunniv2

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/hooklet"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/oracle"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
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

	writeObservationOnce *sync.Once
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
	hook.writeObservationOnce = new(sync.Once)

	return hook
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h

	cloned.Slot0.SqrtPriceX96 = h.Slot0.SqrtPriceX96.Clone()

	cloned.BunniState.RawBalance0 = h.BunniState.RawBalance0.Clone()
	cloned.BunniState.RawBalance1 = h.BunniState.RawBalance1.Clone()
	cloned.BunniState.Reserve0 = h.BunniState.Reserve0.Clone()
	cloned.BunniState.Reserve1 = h.BunniState.Reserve1.Clone()

	cloned.VaultSharePrices.SharedPrice0 = h.VaultSharePrices.SharedPrice0.Clone()
	cloned.VaultSharePrices.SharedPrice1 = h.VaultSharePrices.SharedPrice1.Clone()

	cloned.PoolManagerReserves[0] = h.PoolManagerReserves[0].Clone()
	cloned.PoolManagerReserves[1] = h.PoolManagerReserves[1].Clone()

	cloned.writeObservationOnce = new(sync.Once)

	return &cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	if swapInfo == nil {
		return
	}

	si, ok := swapInfo.(SwapInfo)
	if !ok {
		return
	}

	updateIfNotNil := func(old *uint256.Int, new *uint256.Int) {
		if new != nil {
			old.Set(new)
		}
	}

	updateIfNotNil(h.BunniState.RawBalance0, si.newRawBalance0)
	updateIfNotNil(h.BunniState.RawBalance1, si.newRawBalance1)
	updateIfNotNil(h.BunniState.Reserve0, si.newReserve0)
	updateIfNotNil(h.BunniState.Reserve1, si.newReserve1)
	updateIfNotNil(h.VaultSharePrices.SharedPrice0, si.newVaultSharePrice0)
	updateIfNotNil(h.VaultSharePrices.SharedPrice1, si.newVaultSharePrice1)
	updateIfNotNil(h.PoolManagerReserves[0], si.newPoolManagerReserve0)
	updateIfNotNil(h.PoolManagerReserves[1], si.newPoolManagerReserve1)

	if si.newSlot0 != (Slot0{}) {
		h.Slot0 = si.newSlot0
	}

	if h.LdfState != si.newLdfState {
		h.LdfState = si.newLdfState
	}

	if h.BunniState.IdleBalance != si.newIdleBalance {
		h.BunniState.IdleBalance = si.newIdleBalance
	}
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.ldf == nil {
		return nil, errors.New("ldf is not initialized")
	}

	if h.hooklet == nil {
		return nil, errors.New("hooklet is not initialized")
	}

	amountSpecified := uint256.MustFromBig(params.AmountSpecified)

	if h.BlockTimestamp == 0 {
		h.BlockTimestamp = uint32(time.Now().Unix())
	}

	feeOverridden, feeOverride, priceOverridden, sqrtPriceX96Override :=
		h.hooklet.BeforeSwap(&hooklet.SwapParams{
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

	sqrtPriceLimitX96 := lo.Ternary(
		params.ZeroForOne,
		new(uint256.Int).AddUint64(v3Utils.MinSqrtRatioU256, 1),
		new(uint256.Int).SubUint64(v3Utils.MaxSqrtRatioU256, 1),
	)

	if h.Slot0.SqrtPriceX96.IsZero() ||
		(params.ZeroForOne && sqrtPriceLimitX96.Cmp(h.Slot0.SqrtPriceX96) >= 0) ||
		(!params.ZeroForOne && sqrtPriceLimitX96.Cmp(h.Slot0.SqrtPriceX96) <= 0) ||
		params.AmountSpecified.Cmp(bignumber.MAX_INT_128) > 0 ||
		params.AmountSpecified.Cmp(bignumber.MIN_INT_128) < 0 {
		return nil, errors.New("BunniHook__InvalidSwap")
	}

	var balance0 uint256.Int
	balance0.Add(
		h.BunniState.RawBalance0,
		getReservesInUnderlying(h.Vaults[0], h.BunniState.Reserve0),
	)

	var balance1 uint256.Int
	balance1.Add(
		h.BunniState.RawBalance1,
		getReservesInUnderlying(h.Vaults[1], h.BunniState.Reserve1),
	)

	h.updateOracle()

	useLDFTwap := h.BunniState.TwapSecondsAgo != 0
	useFeeTwap := !feeOverridden && h.HookParams.FeeTwapSecondsAgo != 0

	var (
		arithmeticMeanTick int64
		feeMeanTick        int64
		err                error
	)

	if useLDFTwap && useFeeTwap {
		tickCumulatives, err := h.oracle.ObserveTriple(
			h.ObservationState.IntermediateObservation,
			h.BlockTimestamp,
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
		arithmeticMeanTick, err = h.getTwap()
		if err != nil {
			return nil, err
		}
	} else if useFeeTwap {
		feeMeanTick, err = h.getTwap()
		if err != nil {
			return nil, err
		}
	}

	ldfState := lo.Ternary(
		h.BunniState.LdfType == DYNAMIC_AND_STATEFUL,
		h.LdfState,
		[32]byte{},
	)

	totalLiquidity, _, _, liquidityDensityOfRoundedTickX96, currentActiveBalance0, currentActiveBalance1,
		newLdfState, shouldSurge, err := h.queryLDF(
		h.Slot0.SqrtPriceX96,
		h.Slot0.Tick,
		int(arithmeticMeanTick),
		ldfState,
		&balance0,
		&balance1,
		h.BunniState.IdleBalance,
	)
	if err != nil {
		return nil, err
	}

	var swapInfo SwapInfo

	if (params.ZeroForOne && currentActiveBalance1.IsZero()) ||
		(!params.ZeroForOne && currentActiveBalance0.IsZero()) {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecific:   params.AmountSpecified,
			DeltaUnSpecific: bignumber.ZeroBI,
			SwapInfo:        swapInfo,
		}, nil
	}

	if totalLiquidity.IsZero() ||
		(!params.ExactIn &&
			lo.Ternary(params.ZeroForOne, currentActiveBalance1, currentActiveBalance0).Lt(amountSpecified)) {
		return nil, errors.New("BunniHook__RequestedOutputExceedsBalance")
	}

	shouldSurge = shouldSurge && h.BunniState.LdfType != STATIC

	if h.BunniState.LdfType == DYNAMIC_AND_STATEFUL {
		swapInfo.newLdfState = newLdfState
	}

	if shouldSurge {
		newIdleBalance, err := h.computeIdleBalance(
			currentActiveBalance0,
			currentActiveBalance1,
			&balance0,
			&balance1,
		)
		if err != nil {
			return nil, err
		}
		swapInfo.newIdleBalance = newIdleBalance
	}

	var shouldSurgeFromVaults bool
	shouldSurgeFromVaults, swapInfo.newVaultSharePrice0, swapInfo.newVaultSharePrice1, err = h.shouldSurgeFromVaults()
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
		LdfState:                         ldfState,
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
		timeSinceLastSwap := h.BlockTimestamp - h.Slot0.LastSwapTimestamp
		surgeFeeAutostartThreshold := uint32(h.HookParams.SurgeFeeAutostartThreshold)

		if timeSinceLastSwap >= surgeFeeAutostartThreshold {
			lastSurgeTimestamp = h.Slot0.LastSwapTimestamp + surgeFeeAutostartThreshold
		} else {
			lastSurgeTimestamp = h.BlockTimestamp
		}
	}

	swapInfo.newSlot0 = Slot0{
		SqrtPriceX96:       updatedSqrtPriceX96,
		Tick:               updatedTick,
		LastSurgeTimestamp: lastSurgeTimestamp,
		LastSwapTimestamp:  h.BlockTimestamp,
	}

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
		surgeFee, err := computeSurgeFee(
			h.BlockTimestamp,
			lastSurgeTimestamp,
			h.HookParams.SurgeFeeHalfLife,
		)
		if err != nil {
			return nil, err
		}
		hookFeesBaseSwapFee.Set(u256.Max(feeOverride, surgeFee))
	} else {
		dynamicFee, err := computeDynamicSwapFee(
			h.BlockTimestamp,
			updatedSqrtPriceX96,
			int(feeMeanTick),
			lastSurgeTimestamp,
			h.HookParams.FeeMin,
			h.HookParams.FeeMax,
			h.HookParams.FeeQuadraticMultiplier,
			h.HookParams.SurgeFeeHalfLife,
		)
		if err != nil {
			return nil, err
		}
		hookFeesBaseSwapFee.Set(dynamicFee)
	}

	var (
		swapFee                    *uint256.Int
		swapFeeAmount              *uint256.Int
		hookFeesAmount             *uint256.Int
		curatorFeeAmount           *uint256.Int
		hookHandleSwapInputAmount  uint256.Int
		hookHandleSwapOutputAmount uint256.Int
	)

	result := uniswapv4.BeforeSwapResult{
		Gas: _BEFORE_SWAP_GAS,
	}

	for _, vault := range h.Vaults {
		if !valueobject.IsZeroAddress(vault.Address) {
			result.Gas += _PREVIEW_REDEEM_GAS
		}
	}

	if useAmAmmFee {
		surgeFee, err := computeSurgeFee(
			h.BlockTimestamp,
			lastSurgeTimestamp,
			h.HookParams.SurgeFeeHalfLife,
		)
		if err != nil {
			return nil, err
		}
		swapFee = u256.Max(&amAmmSwapFee, surgeFee)
	} else {
		swapFee = &hookFeesBaseSwapFee
	}

	if params.ExactIn {
		swapFeeAmount = math.MulDivUp(outputAmount, swapFee, SWAP_FEE_BASE)

		if useAmAmmFee {
			baseSwapFeeAmount := math.MulDivUp(
				outputAmount,
				&hookFeesBaseSwapFee,
				SWAP_FEE_BASE,
			)
			hookFeesAmount = math.MulDivUp(baseSwapFeeAmount, h.HookFee, MODIFIER_BASE)
			curatorFeeAmount = math.MulDivUp(
				baseSwapFeeAmount,
				h.CuratorFees.FeeRate,
				CURATOR_FEE_BASE,
			)

			if swapFee.Cmp(&amAmmSwapFee) != 0 {
				swapFeeAdjusted := math.MulDivUp(&hookFeesBaseSwapFee, h.HookFee, MODIFIER_BASE)
				swapFeeAdjusted.Sub(swapFee, swapFeeAdjusted)

				swapFeeAdjusted.Sub(
					swapFeeAdjusted,
					math.MulDivUp(
						&hookFeesBaseSwapFee,
						h.CuratorFees.FeeRate,
						CURATOR_FEE_BASE,
					),
				)
				swapFee = u256.Max(&amAmmSwapFee, swapFeeAdjusted)

				swapFeeAmount = math.MulDivUp(outputAmount, swapFee, SWAP_FEE_BASE)
			}
		} else {
			hookFeesAmount = math.MulDivUp(swapFeeAmount, h.HookFee, MODIFIER_BASE)
			curatorFeeAmount = math.MulDivUp(
				swapFeeAmount,
				h.CuratorFees.FeeRate,
				CURATOR_FEE_BASE,
			)
			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount).Sub(swapFeeAmount, curatorFeeAmount)
		}

		outputAmount.Sub(outputAmount, swapFeeAmount).
			Sub(outputAmount, hookFeesAmount).
			Sub(outputAmount, curatorFeeAmount)

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
		swapFeeAmount = math.MulDivUp(
			inputAmount,
			swapFee,
			new(uint256.Int).Sub(SWAP_FEE_BASE, swapFee),
		)

		if useAmAmmFee {
			baseSwapFeeAmount := math.MulDivUp(
				inputAmount,
				&hookFeesBaseSwapFee,
				new(uint256.Int).Sub(SWAP_FEE_BASE, &hookFeesBaseSwapFee),
			)
			hookFeesAmount = math.MulDivUp(baseSwapFeeAmount, h.HookFee, MODIFIER_BASE)
			curatorFeeAmount = math.MulDivUp(
				baseSwapFeeAmount,
				h.CuratorFees.FeeRate,
				CURATOR_FEE_BASE,
			)
		} else {
			hookFeesAmount = math.MulDivUp(swapFeeAmount, h.HookFee, MODIFIER_BASE)
			curatorFeeAmount = math.MulDivUp(
				swapFeeAmount,
				h.CuratorFees.FeeRate,
				CURATOR_FEE_BASE,
			)
			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount).Sub(swapFeeAmount, curatorFeeAmount)
		}

		inputAmount.Add(inputAmount, swapFeeAmount).
			Add(inputAmount, hookFeesAmount).
			Add(inputAmount, curatorFeeAmount)
		result.DeltaUnSpecific = inputAmount.ToBig()

		actualOutputAmount := u256.Min(amountSpecified, outputAmount).ToBig()
		result.DeltaSpecific = actualOutputAmount.Neg(actualOutputAmount)

		hookHandleSwapOutputAmount.Set(outputAmount)
		hookHandleSwapInputAmount.Sub(inputAmount, hookFeesAmount)

		if useAmAmmFee {
			hookHandleSwapInputAmount.Sub(&hookHandleSwapInputAmount, swapFeeAmount)
		}
	}

	swapInfo.newRawBalance0, swapInfo.newRawBalance1,
		swapInfo.newReserve0, swapInfo.newReserve1,
		swapInfo.newPoolManagerReserve0, swapInfo.newPoolManagerReserve1, err =
		h.hookHandleSwap(
			params.ZeroForOne,
			&hookHandleSwapInputAmount,
			&hookHandleSwapOutputAmount,
			shouldSurge,
		)
	if err != nil {
		return nil, err
	}

	rebalanceOrderDeadline := h.RebalanceOrderDeadline
	if shouldSurge {
		rebalanceOrderDeadline = 0
	}

	if h.HookParams.RebalanceThreshold != 0 &&
		(shouldSurge || h.BlockTimestamp > rebalanceOrderDeadline && rebalanceOrderDeadline != 0) {
		if shouldSurge {
			h.RebalanceOrderDeadline = 0
		}
		result.Gas += _REBALANCE_GAS
	}

	h.hooklet.AfterSwap(nil)
	result.SwapInfo = swapInfo

	return &result, nil
}

func (h *Hook) hookHandleSwap(
	zeroForOne bool,
	inputAmount,
	outputAmount *uint256.Int,
	shouldSurge bool,
) (
	newRawBalance0 *uint256.Int,
	newRawBalance1 *uint256.Int,
	newReserve0 *uint256.Int,
	newReserve1 *uint256.Int,
	newPoolManagerReserve0 *uint256.Int,
	newPoolManagerReserve1 *uint256.Int,
	err error,
) {
	newRawBalance0 = h.BunniState.RawBalance0.Clone()
	newRawBalance1 = h.BunniState.RawBalance1.Clone()
	newReserve0 = h.BunniState.Reserve0.Clone()
	newReserve1 = h.BunniState.Reserve1.Clone()
	newPoolManagerReserve0 = h.PoolManagerReserves[0].Clone()
	newPoolManagerReserve1 = h.PoolManagerReserves[1].Clone()

	if !inputAmount.IsZero() {
		if zeroForOne {
			newRawBalance0.Add(newRawBalance0, inputAmount)
		} else {
			newRawBalance1.Add(newRawBalance1, inputAmount)
		}
	}

	if !outputAmount.IsZero() {
		outputRawBalance, vaultIndex := newRawBalance0.Clone(), 0
		if zeroForOne {
			outputRawBalance, vaultIndex = newRawBalance1.Clone(), 1
		}

		outputVault := h.Vaults[vaultIndex]

		if !valueobject.IsZeroAddress(outputVault.Address) && outputRawBalance.Lt(outputAmount) {
			delta := i256.SafeToInt256(outputRawBalance.Sub(outputAmount, outputRawBalance))

			reserveChange, rawBalanceChange, newPoolManagerReserve, err :=
				h.updateVaultReserveViaClaimTokens(vaultIndex, delta)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}

			if vaultIndex == 0 {
				newPoolManagerReserve0.Set(newPoolManagerReserve)
			} else {
				newPoolManagerReserve1.Set(newPoolManagerReserve)
			}

			if zeroForOne {
				newReserve1 = updateBalance(newReserve1, reserveChange)
				newRawBalance1 = updateBalance(newRawBalance1, rawBalanceChange)
			} else {
				newReserve0 = updateBalance(newReserve0, reserveChange)
				newRawBalance0 = updateBalance(newRawBalance0, rawBalanceChange)
			}

		}

		if zeroForOne {
			newRawBalance1.Sub(newRawBalance1, outputAmount)
		} else {
			newRawBalance0.Sub(newRawBalance0, outputAmount)
		}
	}

	if !shouldSurge {
		if !valueobject.IsZeroAddress(h.Vaults[0].Address) {
			newReserve0, newRawBalance0, newPoolManagerReserve0, err = h.updateRawBalanceIfNeeded(
				0,
				newRawBalance0,
				newReserve0,
				h.BunniState.MinRawTokenRatio0,
				h.BunniState.MaxRawTokenRatio0,
				h.BunniState.TargetRawTokenRatio0,
			)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
		}

		if !valueobject.IsZeroAddress(h.Vaults[1].Address) {
			newReserve1, newRawBalance1, newPoolManagerReserve1, err = h.updateRawBalanceIfNeeded(
				1,
				newRawBalance1,
				newReserve1,
				h.BunniState.MinRawTokenRatio1,
				h.BunniState.MaxRawTokenRatio1,
				h.BunniState.TargetRawTokenRatio1,
			)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
		}
	}

	return newRawBalance0, newRawBalance1, newReserve0, newReserve1, newPoolManagerReserve0, newPoolManagerReserve1, nil
}

func (h *Hook) updateRawBalanceIfNeeded(
	vaultIndex int,
	rawBalance,
	reserve,
	minRatio,
	maxRatio,
	targetRatio *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	var poolManagerReserve = h.PoolManagerReserves[vaultIndex].Clone()

	reserveInUnderlying := getReservesInUnderlying(h.Vaults[vaultIndex], reserve)

	var balance uint256.Int
	balance.Add(rawBalance, reserveInUnderlying)

	maxRawBalance := math.MulDiv(&balance, maxRatio, RAW_TOKEN_RATIO_BASE)
	minRawBalance := math.MulDiv(&balance, minRatio, RAW_TOKEN_RATIO_BASE)

	if rawBalance.Lt(minRawBalance) || rawBalance.Gt(maxRawBalance) {
		targetRawBalance := math.MulDiv(&balance, targetRatio, RAW_TOKEN_RATIO_BASE)

		delta := i256.SafeToInt256(targetRawBalance)
		delta.Sub(delta, i256.SafeToInt256(rawBalance))

		var (
			reserveChange    *int256.Int
			rawBalanceChange *int256.Int
			err              error
		)

		reserveChange, rawBalanceChange, poolManagerReserve, err =
			h.updateVaultReserveViaClaimTokens(vaultIndex, delta)
		if err != nil {
			return nil, nil, nil, err
		}

		newReserveInt := i256.SafeToInt256(reserve)
		newRawBalanceInt := i256.SafeToInt256(rawBalance)

		return i256.SafeConvertToUInt256(newReserveInt.Add(newReserveInt, reserveChange)),
			i256.SafeConvertToUInt256(newRawBalanceInt.Add(newRawBalanceInt, rawBalanceChange)),
			poolManagerReserve,
			nil
	}

	return reserve, rawBalance, poolManagerReserve, nil
}

func updateBalance(balance *uint256.Int, delta *int256.Int) *uint256.Int {
	balanceInt := i256.SafeToInt256(balance)
	balanceInt.Add(balanceInt, delta)
	return i256.SafeConvertToUInt256(balanceInt)
}

func getReservesInUnderlying(vault Vault, reserveAmount *uint256.Int) *uint256.Int {
	if valueobject.IsZeroAddress(vault.Address) {
		return reserveAmount
	}
	return math.MulDivUp(reserveAmount, vault.RedeemRate, WAD)
}

func (h *Hook) updateVaultReserveViaClaimTokens(
	index int,
	rawBalanceChange *int256.Int,
) (*int256.Int, *int256.Int, *uint256.Int, error) {
	absAmount := math.Abs(rawBalanceChange)
	poolManagerReserve := h.PoolManagerReserves[index].Clone()

	var (
		reserveChange          int256.Int
		actualRawBalanceChange int256.Int
	)

	if rawBalanceChange.Sign() < 0 {
		absAmount.Set(u256.Min(
			u256.Min(absAmount, h.Vaults[index].MaxDeposit),
			poolManagerReserve,
		))

		if absAmount.IsZero() {
			return &reserveChange, int256.NewInt(0), poolManagerReserve, nil
		}

		if absAmount.Gt(h.Vaults[index].MaxDeposit) {
			return nil, nil, nil, errors.New("DepositMoreThanMax")
		}

		poolManagerReserve.Sub(poolManagerReserve, absAmount)

		depositedAmt := math.MulDivUp(absAmount, h.Vaults[index].DepositRate, WAD)
		reserveChange.Set(i256.SafeToInt256(depositedAmt))

		actualRawBalanceChange.Set(i256.SafeToInt256(absAmount))
		actualRawBalanceChange.Neg(&actualRawBalanceChange)

	} else if rawBalanceChange.Sign() > 0 {
		if absAmount.Gt(h.Vaults[index].MaxWithdraw) {
			return nil, nil, nil, errors.New("WithdrawMoreThanMax")
		}

		withdrawAmt := math.MulDivUp(absAmount, h.Vaults[index].WithdrawRate, WAD)
		withdrawAmtInt := i256.SafeToInt256(withdrawAmt)

		reserveChange.Set(withdrawAmtInt.Neg(withdrawAmtInt))

		poolManagerReserve.Add(poolManagerReserve, absAmount)

		actualRawBalanceChange.Set(i256.SafeToInt256(absAmount))
	}

	return &reserveChange, &actualRawBalanceChange, poolManagerReserve, nil
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

	var updatedRoundedTickLiquidity uint256.Int
	updatedRoundedTickLiquidity.Mul(input.TotalLiquidity, input.LiquidityDensityOfRoundedTickX96)
	updatedRoundedTickLiquidity.Rsh(&updatedRoundedTickLiquidity, 96)

	updatedTick := h.Slot0.Tick
	sqrtPriceLimitX96 := input.SqrtPriceLimitX96.Clone()

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

		naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut, err = math.ComputeSwapStep(
			input.ExactIn,
			input.ZeroForOne,
			h.Slot0.SqrtPriceX96,
			math.GetSqrtPriceTarget(input.ZeroForOne, sqrtPriceNextX96, sqrtPriceLimitX96),
			&updatedRoundedTickLiquidity,
			input.AmountSpecified,
			u256.U0,
		)
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

			currentBalance := lo.Ternary(
				input.ZeroForOne,
				input.CurrentActiveBalance1,
				input.CurrentActiveBalance0,
			)
			naiveSwapAmountOut = u256.Min(naiveSwapAmountOut, currentBalance)

			return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
		}
	}

	var inverseCumulativeAmountFnInput uint256.Int
	if input.ExactIn {
		inverseCumulativeAmountFnInput.Set(lo.Ternary(
			input.ZeroForOne,
			input.CurrentActiveBalance0,
			input.CurrentActiveBalance1,
		))
		inverseCumulativeAmountFnInput.Add(&inverseCumulativeAmountFnInput, &inputAmount)
	} else {
		inverseCumulativeAmountFnInput.Set(lo.Ternary(
			input.ZeroForOne,
			input.CurrentActiveBalance1,
			input.CurrentActiveBalance0,
		))
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
		input.LdfState,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	if success {
		if (input.ZeroForOne && updatedRoundedTick >= roundedTick) ||
			(!input.ZeroForOne && updatedRoundedTick <= roundedTick) {

			if updatedRoundedTickLiquidity.IsZero() {
				return h.Slot0.SqrtPriceX96, h.Slot0.Tick, u256.U0, u256.U0, nil
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

			currentBalance := lo.Ternary(
				input.ZeroForOne,
				input.CurrentActiveBalance1,
				input.CurrentActiveBalance0,
			)
			naiveSwapAmountOut = u256.Min(naiveSwapAmountOut, currentBalance)

			return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
		}

		tickStart := updatedRoundedTick
		tickNext := updatedRoundedTick + h.tickSpacing
		if input.ZeroForOne {
			tickStart, tickNext = tickNext, tickStart
		}

		startSqrtPriceX96, err := math.GetSqrtPriceAtTick(tickStart)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		if input.ZeroForOne && sqrtPriceLimitX96.Lt(startSqrtPriceX96) ||
			(!input.ZeroForOne && sqrtPriceLimitX96.Gt(startSqrtPriceX96)) {

			sqrtPriceNextX96, err := math.GetSqrtPriceAtTick(tickNext)
			if err != nil {
				return nil, 0, nil, nil, err
			}

			if input.ZeroForOne {
				cumulativeAmount0.Set(u256.Max(cumulativeAmount0, input.CurrentActiveBalance0))
			} else {
				cumulativeAmount1.Set(u256.Max(cumulativeAmount1, input.CurrentActiveBalance1))
			}

			var hitSqrtPriceLimit bool
			if swapLiquidity.IsZero() || sqrtPriceLimitX96.Eq(startSqrtPriceX96) {
				naiveSwapResultSqrtPriceX96 = startSqrtPriceX96.Clone()
				naiveSwapAmountIn = u256.U0
				naiveSwapAmountOut = u256.U0
			} else {
				var amountSpecifiedRemaining uint256.Int

				if input.ExactIn {
					amountSpecifiedRemaining.Sub(
						&inverseCumulativeAmountFnInput,
						lo.Ternary(input.ZeroForOne, cumulativeAmount0, cumulativeAmount1),
					)
				} else {
					amountSpecifiedRemaining.Sub(
						lo.Ternary(input.ZeroForOne, cumulativeAmount1, cumulativeAmount0),
						&inverseCumulativeAmountFnInput,
					)
				}

				naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut, err = math.ComputeSwapStep(
					input.ExactIn,
					input.ZeroForOne,
					startSqrtPriceX96,
					math.GetSqrtPriceTarget(input.ZeroForOne, sqrtPriceNextX96, sqrtPriceLimitX96),
					swapLiquidity,
					&amountSpecifiedRemaining,
					EPSILON_FEE,
				)
				if err != nil {
					return nil, 0, nil, nil, err
				}

				if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceLimitX96) && !sqrtPriceLimitX96.Eq(sqrtPriceNextX96) {
					hitSqrtPriceLimit = true
				}
			}

			if !hitSqrtPriceLimit {
				updatedTick = tickStart

				if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceNextX96) {
					updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
				} else if !naiveSwapResultSqrtPriceX96.Eq(startSqrtPriceX96) {
					updatedTick, err = math.GetTickAtSqrtPrice(naiveSwapResultSqrtPriceX96)
					if err != nil {
						return nil, 0, nil, nil, err
					}
				}

				updatedSqrtPriceX96 := naiveSwapResultSqrtPriceX96.Clone()

				if (input.ExactIn && naiveSwapAmountIn.Eq(input.AmountSpecified)) ||
					(!input.ExactIn && naiveSwapAmountOut.Eq(input.AmountSpecified)) {

					currentBalance := lo.Ternary(
						input.ZeroForOne,
						input.CurrentActiveBalance1,
						input.CurrentActiveBalance0,
					)
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

				if input.ZeroForOne {
					inputAmount.Sub(&updatedActiveBalance0, input.CurrentActiveBalance0)
					outputAmount.Set(math.SubReLU(input.CurrentActiveBalance1, &updatedActiveBalance1))
				} else {
					inputAmount.Sub(&updatedActiveBalance1, input.CurrentActiveBalance1)
					outputAmount.Set(math.SubReLU(input.CurrentActiveBalance0, &updatedActiveBalance0))
				}

				return updatedSqrtPriceX96, updatedTick, &inputAmount, &outputAmount, nil
			}
		}
	}

	updatedSqrtPriceX96 := sqrtPriceLimitX96.Clone()

	if sqrtPriceLimitX96.Eq(h.Slot0.SqrtPriceX96) {
		updatedTick = h.Slot0.Tick
	} else {
		updatedTick, err = math.GetTickAtSqrtPrice(sqrtPriceLimitX96)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}

	_, totalDensity0X96, totalDensity1X96, _, _, _, _, _, err := h.queryLDF(
		updatedSqrtPriceX96,
		updatedTick,
		input.ArithmeticMeanTick,
		input.LdfState,
		u256.U0,
		u256.U0,
		ZERO_BALANCE,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	updatedActiveBalance0, err := math.FullMulX96Up(totalDensity0X96, input.TotalLiquidity)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	updatedActiveBalance1, err := math.FullMulX96Up(totalDensity1X96, input.TotalLiquidity)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	if input.ZeroForOne {
		inputAmount.Sub(updatedActiveBalance0, input.CurrentActiveBalance0)
		outputAmount.Set(math.SubReLU(input.CurrentActiveBalance1, updatedActiveBalance1))
	} else {
		inputAmount.Sub(updatedActiveBalance1, input.CurrentActiveBalance1)
		outputAmount.Set(math.SubReLU(input.CurrentActiveBalance0, updatedActiveBalance0))
	}

	return updatedSqrtPriceX96, updatedTick, &inputAmount, &outputAmount, nil
}

func (h *Hook) queryLDF(
	sqrtPriceX96 *uint256.Int,
	tick,
	arithmeticMeanTick int,
	ldfState [32]byte,
	balance0,
	balance1 *uint256.Int,
	idleBalance [32]byte,
) (
	totalLiquidity,
	totalDensity0X96,
	totalDensity1X96,
	liquidityDensityOfRoundedTickX96,
	activeBalance0,
	activeBalance1 *uint256.Int,
	newLdfState [32]byte,
	shouldSurge bool,
	err error,
) {
	roundedTick, nextRoundedTick := math.RoundTick(h.Slot0.Tick, h.tickSpacing)

	roundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, ldfState, false, err
	}
	nextRoundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(nextRoundedTick)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, ldfState, false, err
	}

	liquidityDensityOfRoundedTickX96, density0RightOfRoundedTickX96, density1LeftOfRoundedTickX96,
		newLdfState, shouldSurge, err := h.ldf.Query(
		roundedTick,
		arithmeticMeanTick,
		tick,
		h.BunniState.LdfParams,
		ldfState,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, ldfState, false, err
	}

	density0OfRoundedTickX96, density1OfRoundedTickX96, err := getAmountsForLiquidity(
		sqrtPriceX96,
		roundedTickSqrtRatio,
		nextRoundedTickSqrtRatio,
		liquidityDensityOfRoundedTickX96,
		true,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, ldfState, false, err
	}

	totalDensity0X96 = density0OfRoundedTickX96.Add(density0RightOfRoundedTickX96, density0OfRoundedTickX96)
	totalDensity1X96 = density1OfRoundedTickX96.Add(density1LeftOfRoundedTickX96, density1OfRoundedTickX96)

	modifiedBalance0 := balance0.Clone()
	modifiedBalance1 := balance1.Clone()

	if !shouldSurge {
		idleBalance, isToken0 := math.FromIdleBalance(idleBalance)
		if isToken0 {
			modifiedBalance0 = math.SubReLU(modifiedBalance0, idleBalance)
		} else {
			modifiedBalance1 = math.SubReLU(modifiedBalance1, idleBalance)
		}
	}

	if !modifiedBalance0.IsZero() || !modifiedBalance1.IsZero() {
		noToken0 := modifiedBalance0.IsZero() || totalDensity0X96.IsZero()
		noToken1 := modifiedBalance1.IsZero() || totalDensity1X96.IsZero()

		var totalLiquidityEstimate0, totalLiquidityEstimate1 *uint256.Int

		if noToken0 {
			totalLiquidityEstimate0 = u256.U0
		} else {
			totalLiquidityEstimate0, err = math.FullMulDiv(modifiedBalance0, Q96, totalDensity0X96)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, ldfState, false, err
			}
		}

		if noToken1 {
			totalLiquidityEstimate1 = u256.U0
		} else {
			totalLiquidityEstimate1, err = math.FullMulDiv(modifiedBalance1, Q96, totalDensity1X96)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, ldfState, false, err
			}
		}

		useLiquidityEstimate0 := (totalLiquidityEstimate0.Lt(totalLiquidityEstimate1) ||
			totalDensity1X96.IsZero()) && !totalDensity0X96.IsZero()

		if useLiquidityEstimate0 {
			if noToken0 {
				totalLiquidity = u256.U0
				activeBalance0 = u256.U0
			} else {
				totalLiquidity, err = math.RoundUpFullMulDivResult(
					modifiedBalance0,
					Q96,
					totalDensity0X96,
					totalLiquidityEstimate0,
				)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}

				temp, err := math.FullMulX96(totalLiquidityEstimate0, totalDensity0X96)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance0 = u256.Min(modifiedBalance0, temp)
			}

			if noToken1 {
				activeBalance1 = u256.U0
			} else {
				temp, err := math.FullMulX96(totalLiquidityEstimate0, totalDensity1X96)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance1 = u256.Min(modifiedBalance1, temp)
			}
		} else {
			if noToken1 {
				totalLiquidity = u256.U0
				activeBalance1 = u256.U0
			} else {
				totalLiquidity, err = math.RoundUpFullMulDivResult(
					modifiedBalance1,
					Q96,
					totalDensity1X96,
					totalLiquidityEstimate1,
				)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}

				temp, err := math.FullMulX96(totalLiquidityEstimate1, totalDensity1X96)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance1 = u256.Min(modifiedBalance1, temp)
			}

			if noToken0 {
				activeBalance0 = u256.U0
			} else {
				temp, err := math.FullMulX96(totalLiquidityEstimate1, totalDensity0X96)
				if err != nil {
					return nil, nil, nil, nil, nil, nil, ldfState, false, err
				}
				activeBalance0 = u256.Min(modifiedBalance0, temp)
			}
		}
	}

	return totalLiquidity, totalDensity0X96, totalDensity1X96, liquidityDensityOfRoundedTickX96,
		activeBalance0, activeBalance1, newLdfState, shouldSurge, nil
}

func getAmountsForLiquidity(
	sqrtPriceX96,
	sqrtPriceAX96,
	sqrtPriceBX96,
	liquidity *uint256.Int,
	roundUp bool,
) (*uint256.Int, *uint256.Int, error) {
	if sqrtPriceAX96.Gt(sqrtPriceBX96) {
		sqrtPriceAX96, sqrtPriceBX96 = sqrtPriceBX96, sqrtPriceAX96
	}

	var (
		amount0, amount1 = new(uint256.Int), new(uint256.Int)
		err              error
	)

	if sqrtPriceX96.Cmp(sqrtPriceAX96) <= 0 {
		amount0, err = math.GetAmount0Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
	} else if sqrtPriceX96.Lt(sqrtPriceBX96) {
		amount0, err = math.GetAmount0Delta(sqrtPriceX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}

		amount1, err = math.GetAmount1Delta(sqrtPriceAX96, sqrtPriceX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
	} else {
		amount1, err = math.GetAmount1Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
	}

	return amount0, amount1, nil
}

func (h *Hook) getTwap() (int64, error) {
	tickCumulatives, err := h.oracle.ObserveDouble(
		h.ObservationState.IntermediateObservation,
		h.BlockTimestamp,
		[]uint32{h.BunniState.TwapSecondsAgo, 0},
		h.Slot0.Tick,
		h.ObservationState.Index,
		h.ObservationState.Cardinality,
	)
	if err != nil {
		return 0, err
	}

	tickCumulativesDelta := tickCumulatives[1] - tickCumulatives[0]
	return tickCumulativesDelta / int64(h.BunniState.TwapSecondsAgo), nil
}

func (h *Hook) updateOracle() {
	h.writeObservationOnce.Do(func() {
		h.ObservationState.IntermediateObservation, h.ObservationState.Index, h.ObservationState.Cardinality =
			h.oracle.Write(
				h.ObservationState.IntermediateObservation,
				h.ObservationState.Index,
				h.BlockTimestamp,
				h.Slot0.Tick,
				h.ObservationState.Cardinality,
				h.ObservationState.CardinalityNext,
				h.HookParams.OracleMinInterval,
			)
	})
}

func computeSurgeFee(
	blockTimestamp,
	lastSurgeTimestamp uint32,
	surgeFeeHalfLife *uint256.Int,
) (fee *uint256.Int, err error) {
	timeSinceLastSurge := uint256.NewInt(uint64(blockTimestamp - lastSurgeTimestamp))

	fee = math.MulDiv(timeSinceLastSurge, LN2_WAD, surgeFeeHalfLife)

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

	quadraticTerm := math.MulDivUp(feeQuadraticMultiplier, delta, SWAP_FEE_BASE_SQUARED)

	return u256.Max(fee, u256.Min(quadraticTerm.Add(quadraticTerm, feeMin), feeMax)), nil
}

func (h *Hook) shouldSurgeFromVaults() (bool, *uint256.Int, *uint256.Int, error) {
	var (
		shouldSurge              bool
		sharePrice0, sharePrice1 uint256.Int
	)

	if !valueobject.IsZeroAddress(h.Vaults[0].Address) || !valueobject.IsZeroAddress(h.Vaults[1].Address) {
		rescaleFactor0 := 18 + h.Vaults[0].Decimals - h.BunniState.Currency0Decimals
		rescaleFactor1 := 18 + h.Vaults[1].Decimals - h.BunniState.Currency1Decimals

		if !h.BunniState.Reserve0.IsZero() {
			reserveBalance0 := getReservesInUnderlying(h.Vaults[0], h.BunniState.Reserve0)
			sharePrice0.Set(math.MulDivUp(
				reserveBalance0,
				u256.TenPow(rescaleFactor0),
				h.BunniState.Reserve0,
			))
		}

		if !h.BunniState.Reserve1.IsZero() {
			reserveBalance1 := getReservesInUnderlying(h.Vaults[1], h.BunniState.Reserve1)
			sharePrice1.Set(math.MulDivUp(
				reserveBalance1,
				u256.TenPow(rescaleFactor1),
				h.BunniState.Reserve1,
			))
		}

		shouldSurge = h.VaultSharePrices.Initialized &&
			(math.Dist(&sharePrice0, h.VaultSharePrices.SharedPrice0).
				Gt(new(uint256.Int).Div(h.VaultSharePrices.SharedPrice0, h.HookParams.VaultSurgeThreshold0)) ||
				math.Dist(&sharePrice1, h.VaultSharePrices.SharedPrice1).
					Gt(new(uint256.Int).Div(h.VaultSharePrices.SharedPrice1, h.HookParams.VaultSurgeThreshold1)))

		if !h.VaultSharePrices.Initialized || !sharePrice0.Eq(h.VaultSharePrices.SharedPrice0) ||
			!sharePrice1.Eq(h.VaultSharePrices.SharedPrice1) {
			h.VaultSharePrices.Initialized = true
		}
	}

	return shouldSurge, &sharePrice0, &sharePrice1, nil
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
