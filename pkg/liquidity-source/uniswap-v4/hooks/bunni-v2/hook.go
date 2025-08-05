package bunniv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/samber/lo"

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

type Hook struct {
	*uniswapv4.BaseHook

	hook            common.Address
	hookletAddress  string
	hooklet         hooklet.IHooklet
	slot0           Slot0
	bunniState      PoolState
	ldfState        [32]byte
	prevSharePrices VaultSharePrices

	state               ObservationState
	observation         *oracle.ObservationStorage
	hookParams          DecodedHookParams
	amAmm               AmAmm
	env                 Env
	curatorFee          CuratorFee
	ldf                 ldf.ILiquidityDensityFunction
	vaults              [2]Vault
	reserveBalances     [2]*uint256.Int
	poolManagerReserves [2]*uint256.Int
	isNative            [2]bool
	tickSpacing         int

	rebalanceOrderDeadline *uint256.Int
}

type Vault struct {
	Address    common.Address
	RedeemRate *uint256.Int
	MaxDeposit *uint256.Int
}

type VaultSharePrices struct {
	Initialized  bool
	SharedPrice0 *uint256.Int
	SharedPrice1 *uint256.Int
}

type CuratorFee struct {
	FeeRate *uint256.Int
}
type Env struct {
	HookFeeModifier *uint256.Int
}

type AmAmm struct {
	AmAmmManager common.Address
	SwapFee0For1 *uint256.Int
	SwapFee1For0 *uint256.Int
}

type HookExtra struct {
	HookletAddress string
	HookletExtra   string
	LDFAddress     string
}

type DecodedHookParams struct {
	FeeMin                     *uint256.Int
	FeeMax                     *uint256.Int
	FeeQuadraticMultiplier     *uint256.Int
	FeeTwapSecondsAgo          uint32
	MaxAmAmmFee                *uint256.Int
	SurgeFeeHalfLife           *uint256.Int
	SurgeFeeAutostartThreshold uint16
	VaultSurgeThreshold0       *uint256.Int
	VaultSurgeThreshold1       *uint256.Int
	RebalanceThreshold         uint16
	RebalanceMaxSlippage       uint16
	RebalanceTwapSecondsAgo    uint16
	RebalanceOrderTTL          uint16
	AmAmmEnabled               bool
	OracleMinInterval          uint32
	MinRentMultiplier          *uint256.Int
}

type Slot0 struct {
	SqrtPriceX96       *uint256.Int
	Tick               int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}

type PoolState struct {
	LiquidityDensityFunction common.Address
	BunniToken               common.Address
	Hooklet                  common.Address
	TwapSecondsAgo           uint32
	LdfParams                [32]byte
	HookParams               []byte
	LdfType                  uint8
	MinRawTokenRatio0        *uint256.Int
	TargetRawTokenRatio0     *uint256.Int
	MaxRawTokenRatio0        *uint256.Int
	MinRawTokenRatio1        *uint256.Int
	TargetRawTokenRatio1     *uint256.Int
	MaxRawTokenRatio1        *uint256.Int
	Currency0Decimals        uint8
	Currency1Decimals        uint8
	Vault0Decimals           uint8
	Vault1Decimals           uint8
	RawBalance0              *uint256.Int
	RawBalance1              *uint256.Int
	Reserve0                 *uint256.Int
	Reserve1                 *uint256.Int
	IdleBalance              [32]byte
}

type Slot0RPC struct {
	SqrtPriceX96       *big.Int
	Tick               *big.Int
	LastSwapTimestamp  uint32
	LastSurgeTimestamp uint32
}

type VaultRPC struct {
	ReserveBalance *big.Int
}

type PoolStateRPC struct {
	Data struct {
		LiquidityDensityFunction common.Address
		BunniToken               common.Address
		Hooklet                  common.Address
		TwapSecondsAgo           *big.Int
		LdfParams                [32]byte
		HookParams               []byte
		Vault0                   common.Address
		Vault1                   common.Address
		LdfType                  uint8
		MinRawTokenRatio0        *big.Int
		TargetRawTokenRatio0     *big.Int
		MaxRawTokenRatio0        *big.Int
		MinRawTokenRatio1        *big.Int
		TargetRawTokenRatio1     *big.Int
		MaxRawTokenRatio1        *big.Int
		Currency0Decimals        uint8
		Currency1Decimals        uint8
		Vault0Decimals           uint8
		Vault1Decimals           uint8
		RawBalance0              *big.Int
		RawBalance1              *big.Int
		Reserve0                 *big.Int
		Reserve1                 *big.Int
		IdleBalance              [32]byte
	}
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, lo.Keys(HookAddresses)...)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		hook:     param.HookAddress,
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4BunniV2},
	}

	var hookExtra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &hookExtra); err != nil {
			return nil
		}
	}

	var poolStaticExtra uniswapv4.StaticExtra
	if param.Pool.StaticExtra != "" {
		if err := json.Unmarshal([]byte(param.Pool.StaticExtra), &poolStaticExtra); err != nil {
			return nil
		}
	}

	hook.hookletAddress = hookExtra.HookletAddress

	hook.hooklet = InitHooklet(hook.hookletAddress, hookExtra.HookletExtra)

	hook.isNative = poolStaticExtra.IsNative
	hook.ldf = InitLDF(hookExtra.LDFAddress, int(poolStaticExtra.TickSpacing))
	// if hook.ldf == nil {
	// 	return nil
	// }

	return hook
}

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	// if err := h.hubCallerErr; err != nil {
	// 	return nil, err
	// }

	// poolState, err := h.hubCaller.PoolState(&bind.CallOpts{Context: ctx}, common.HexToHash(param.Pool.Address))
	// if err != nil {
	// 	return nil, err
	// }

	// return entity.PoolReserves{
	// 	poolState.Reserve0.Add(poolState.Reserve0, poolState.RawBalance0).String(),
	// 	poolState.Reserve1.Add(poolState.Reserve1, poolState.RawBalance1).String(),
	// }, nil

	return nil, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var hookExtra HookExtra
	if param.HookExtra != "" {
		if err := json.Unmarshal([]byte(param.HookExtra), &hookExtra); err != nil {
			return "", err
		}
	}

	poolBytes := eth.StringToBytes32(param.Pool.Address)

	var (
		ldfState     [32]byte
		slot0        Slot0RPC
		poolState    PoolStateRPC
		poolBalances [2]*big.Int
		hookParams   []byte
		feeOverride  hooklet.FeeOverride
		hookStates   [4]common.Hash

		reserveBalance0, reserveBalance1 = big.NewInt(0), big.NewInt(0)
	)

	hookCalls := param.RpcClient.NewRequest().SetContext(ctx)

	slotState := crypto.Keccak256Hash(poolBytes[:], common.LeftPadBytes(bignumber.Seven.Bytes(), 32))
	slotObservation := common.BigToHash(new(big.Int).Add(slotState.Big(), bignumber.One))
	slotRebalance := crypto.Keccak256Hash(poolBytes[:], common.LeftPadBytes(bignumber.Eight.Bytes(), 32))
	slotVaultSharePrices := crypto.Keccak256Hash(poolBytes[:], common.LeftPadBytes(VAULT_SHARE_PRICES_SLOT.Bytes(), 32))
	slotCuratorFees := crypto.Keccak256Hash(poolBytes[:], common.LeftPadBytes(CURATOR_FEES_SLOT.Bytes(), 32))
	slotHookFee := crypto.Keccak256Hash(poolBytes[:], common.LeftPadBytes(HOOK_FEE_SLOT.Bytes(), 32))

	hubAddress := GetHubAddress(h.hook)
	hookAddress := h.hook.Hex()

	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "extsload",
		Params: []any{
			[]common.Hash{
				slotState,
				slotObservation,
				slotRebalance,
				slotVaultSharePrices,
				slotCuratorFees,
				slotHookFee,
			},
		},
	}, []any{&hookStates})
	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "ldfStates",
		Params: []any{poolBytes},
	}, []any{&ldfState})
	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHookABI,
		Target: hookAddress,
		Method: "slot0s",
		Params: []any{poolBytes},
	}, []any{&slot0})
	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: hubAddress,
		Method: "poolState",
		Params: []any{poolBytes},
	}, []any{&poolState})
	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: hubAddress,
		Method: "hookParams",
		Params: []any{poolBytes},
	}, []any{&hookParams})
	hookCalls.AddCall(&ethrpc.Call{
		ABI:    bunniHubABI,
		Target: hubAddress,
		Method: "poolBalances",
		Params: []any{poolBytes},
	}, []any{&poolBalances})

	if _, err := hookCalls.Aggregate(); err != nil {
		return "", err
	}

	log.Fatalf("%+v\n", hookStates)

	hookletCalls := param.RpcClient.NewRequest().SetContext(ctx)
	hookletCalls.AddCall(&ethrpc.Call{
		ABI:    feeOverrideHookletABI,
		Target: poolState.Data.Hooklet.Hex(),
		Method: "feeOverrides",
		Params: []any{poolBytes},
	}, []any{&feeOverride})
	hookletCalls.AddCall(&ethrpc.Call{
		ABI:    erc4626ABI,
		Target: poolState.Data.Vault0.Hex(),
		Method: "previewRedeem",
		Params: []any{poolState.Data.Reserve0},
	}, []any{&reserveBalance0})
	hookletCalls.AddCall(&ethrpc.Call{
		ABI:    erc4626ABI,
		Target: poolState.Data.Vault1.Hex(),
		Method: "previewRedeem",
		Params: []any{poolState.Data.Reserve1},
	}, []any{&reserveBalance1})

	if _, err := hookletCalls.Aggregate(); err != nil {
		return "", err
	}

	h.hookParams = DecodeHookParams(hookParams)

	return "", nil
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapHookParams) (*uniswapv4.BeforeSwapHookResult, error) {
	amountSpecified := uint256.MustFromBig(params.AmountSpecified)

	blockTimestamp := uint32(time.Now().Unix())

	feeOverridden, feeOverride, priceOverridden, sqrtPriceX96Override := h.hooklet.BeforeSwap(&hooklet.SwapParams{
		ZeroForOne: params.ZeroForOne,
	})

	// Apply price override if needed
	if priceOverridden {
		h.slot0.SqrtPriceX96.Set(sqrtPriceX96Override)
		var err error
		h.slot0.Tick, err = math.GetTickAtSqrtPrice(sqrtPriceX96Override)
		if err != nil {
			return nil, err
		}
	}

	// Validate swap parameters
	sqrtPriceLimitX96 := uint256.MustFromBig(params.SqrtPriceLimitX96)
	if h.slot0.SqrtPriceX96.IsZero() ||
		(params.ZeroForOne && (sqrtPriceLimitX96.Cmp(h.slot0.SqrtPriceX96) >= 0 ||
			sqrtPriceLimitX96.Cmp(v3Utils.MinSqrtRatioU256) <= 0)) ||
		(!params.ZeroForOne && (sqrtPriceLimitX96.Cmp(h.slot0.SqrtPriceX96) <= 0 ||
			sqrtPriceLimitX96.Cmp(v3Utils.MaxSqrtRatioU256) >= 0)) ||
		params.AmountSpecified.Cmp(bignumber.MAX_INT_128) > 0 ||
		params.AmountSpecified.Cmp(bignumber.MIN_INT_128) < 0 {
		return nil, fmt.Errorf("BunniHook__InvalidSwap")
	}

	// Compute total token balances
	var balance0 uint256.Int
	balance0.Add(h.bunniState.RawBalance0, h.reserveBalances[0])

	var balance1 uint256.Int
	balance1.Add(h.bunniState.RawBalance1, h.reserveBalances[1])

	// Update TWAP oracle
	h.updateOracle(blockTimestamp)

	// Determine TWAP usage
	useLDFTwap := h.bunniState.TwapSecondsAgo != 0
	useFeeTwap := !feeOverridden && h.hookParams.FeeTwapSecondsAgo != 0

	// Calculate TWAP values
	var arithmeticMeanTick int64
	var feeMeanTick int64
	var err error
	if useLDFTwap && useFeeTwap {
		// Get triple observation for both LDF and fee TWAP
		tickCumulatives, err := h.observation.ObserveTriple(
			h.state.IntermediateObservation,
			blockTimestamp,
			[]uint32{0, h.bunniState.TwapSecondsAgo, h.hookParams.FeeTwapSecondsAgo},
			h.slot0.Tick,
			h.state.Index,
			h.state.Cardinality,
		)
		if err != nil {
			return nil, err
		}

		arithmeticMeanTick = (tickCumulatives[0] - tickCumulatives[1]) / int64(h.bunniState.TwapSecondsAgo)
		feeMeanTick = (tickCumulatives[0] - tickCumulatives[2]) / int64(h.hookParams.FeeTwapSecondsAgo) // feeMeanTick
	} else if useLDFTwap {
		arithmeticMeanTick, err = h.getTwap(blockTimestamp)
		if err != nil {
			return nil, err
		}
	} else if useFeeTwap {
		feeMeanTick, err = h.getTwap(blockTimestamp) // feeMeanTick
		if err != nil {
			return nil, err
		}
	}

	// Query LDF for liquidity and token densities
	totalLiquidity, _, currentActiveBalance0, currentActiveBalance1,
		newLdfState, shouldSurge, err := h.queryLDF(h.slot0.SqrtPriceX96, h.slot0.Tick,
		int(arithmeticMeanTick), h.ldfState, &balance0, &balance1, h.bunniState.IdleBalance)
	if err != nil {
		return nil, err
	}

	// Validate output token availability
	if (params.ZeroForOne && currentActiveBalance1.IsZero()) ||
		(!params.ZeroForOne && currentActiveBalance0.IsZero()) ||
		totalLiquidity.IsZero() ||
		(!params.ExactIn &&
			lo.Ternary(params.ZeroForOne, currentActiveBalance1, currentActiveBalance0).Lt(amountSpecified)) {
		return nil, fmt.Errorf("BunniHook__RequestedOutputExceedsBalance")
	}

	shouldSurge = shouldSurge && h.bunniState.LdfType != STATIC

	if h.bunniState.LdfType == DYNAMIC_AND_STATEFUL {
		h.ldfState = newLdfState
	}

	if shouldSurge {
		// todo
	}

	shouldSurgeFromVaults, err := h.shouldSurgeFromVaults()
	if err != nil {
		return nil, err
	}

	shouldSurge = shouldSurge || shouldSurgeFromVaults

	updatedSqrtPriceX96, updatedTick, inputAmount, outputAmount, err := h.computeSwap(BunniComputeSwapInput{})
	if err != nil {
		return nil, err
	}

	if !params.ExactIn && outputAmount.Lt(amountSpecified) {
		return nil, errors.New("BunniHook__InsufficientOutput")
	}

	if (params.ZeroForOne && updatedSqrtPriceX96.Gt(h.slot0.SqrtPriceX96)) ||
		(!params.ZeroForOne && updatedSqrtPriceX96.Lt(h.slot0.SqrtPriceX96)) ||
		(outputAmount.IsZero() || inputAmount.IsZero()) {
		return nil, errors.New("BunniHook__InvalidSwap")
	}

	lastSurgeTimestamp := h.slot0.LastSurgeTimestamp

	if shouldSurge {
		timeSinceLastSwap := blockTimestamp - h.slot0.LastSwapTimestamp

		surgeFeeAutostartThreshold := uint32(h.hookParams.SurgeFeeAutostartThreshold)
		if timeSinceLastSwap >= surgeFeeAutostartThreshold {
			lastSurgeTimestamp = h.slot0.LastSwapTimestamp + surgeFeeAutostartThreshold
		} else {
			lastSurgeTimestamp = blockTimestamp
		}
	}

	h.slot0.SqrtPriceX96 = updatedSqrtPriceX96
	h.slot0.Tick = updatedTick
	h.slot0.LastSwapTimestamp = blockTimestamp
	h.slot0.LastSurgeTimestamp = lastSurgeTimestamp

	var amAmmSwapFee uint256.Int
	if h.hookParams.AmAmmEnabled {
		// todo
		if params.ZeroForOne {
			amAmmSwapFee.Set(h.amAmm.SwapFee0For1)
		} else {
			amAmmSwapFee.Set(h.amAmm.SwapFee1For0)
		}
	}

	useAmAmmFee := h.hookParams.AmAmmEnabled && !valueobject.IsZeroAddress(h.amAmm.AmAmmManager)

	var hookFeesBaseSwapFee uint256.Int
	if feeOverridden {
		surgeFee, err := computeSurgeFee(blockTimestamp, lastSurgeTimestamp, h.hookParams.SurgeFeeHalfLife)
		if err != nil {
			return nil, err
		}

		hookFeesBaseSwapFee.Set(u256.Max(feeOverride, surgeFee))
	} else {
		dynamicFee, err := computeDynamicSwapFee(blockTimestamp, updatedSqrtPriceX96, int(feeMeanTick), lastSurgeTimestamp,
			h.hookParams.FeeMin, h.hookParams.FeeMax, h.hookParams.FeeQuadraticMultiplier, h.hookParams.SurgeFeeHalfLife)
		if err != nil {
			return nil, err
		}

		hookFeesBaseSwapFee.Set(dynamicFee)
	}

	var swapFee uint256.Int
	var swapFeeAmount *uint256.Int
	var hookFeesAmount *uint256.Int
	var curatorFeeAmount *uint256.Int
	var hookHandleSwapInputAmount *uint256.Int
	var hookHandleSwapOutputAmount *uint256.Int

	var result uniswapv4.BeforeSwapHookResult

	if useAmAmmFee {
		surgeFee, err := computeSurgeFee(blockTimestamp, lastSurgeTimestamp, h.hookParams.SurgeFeeHalfLife)
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

			hookFeesAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.env.HookFeeModifier, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.curatorFee.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			if swapFee.Cmp(&amAmmSwapFee) != 0 {
				swapFeeAdjusted, err := v3Utils.MulDivRoundingUp(&hookFeesBaseSwapFee, h.env.HookFeeModifier, MODIFIER_BASE)
				if err != nil {
					return nil, err
				}

				swapFeeAdjusted.Sub(&swapFee, swapFeeAdjusted)

				tmp, err := v3Utils.MulDivRoundingUp(&hookFeesBaseSwapFee, h.curatorFee.FeeRate, CURATOR_FEE_BASE)
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
			hookFeesAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.env.HookFeeModifier, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.curatorFee.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount)
			swapFeeAmount.Sub(swapFeeAmount, curatorFeeAmount)
		}

		outputAmount.Sub(outputAmount, swapFeeAmount)
		outputAmount.Sub(outputAmount, hookFeesAmount)
		outputAmount.Sub(outputAmount, curatorFeeAmount)

		var actualInputAmount uint256.Int
		if amountSpecified.Lt(inputAmount) {
			actualInputAmount.Set(inputAmount)
		}

		result.DeltaSpecific = actualInputAmount.ToBig()
		result.DeltaUnSpecific = outputAmount.ToBig()

		hookHandleSwapInputAmount = inputAmount.Clone()

		var hookHandleSwapOutputAmount uint256.Int
		hookHandleSwapOutputAmount.Add(outputAmount, hookFeesAmount)
		hookHandleSwapOutputAmount.Add(&hookHandleSwapOutputAmount, curatorFeeAmount)

		if useAmAmmFee {
			hookHandleSwapOutputAmount.Add(outputAmount, swapFeeAmount)
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

			hookFeesAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.env.HookFeeModifier, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(baseSwapFeeAmount, h.curatorFee.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}
		} else {
			hookFeesAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.env.HookFeeModifier, MODIFIER_BASE)
			if err != nil {
				return nil, err
			}

			curatorFeeAmount, err = v3Utils.MulDivRoundingUp(swapFeeAmount, h.curatorFee.FeeRate, CURATOR_FEE_BASE)
			if err != nil {
				return nil, err
			}

			swapFeeAmount.Sub(swapFeeAmount, hookFeesAmount)
			swapFeeAmount.Sub(swapFeeAmount, curatorFeeAmount)
		}

		inputAmount.Add(inputAmount, swapFeeAmount)
		inputAmount.Add(inputAmount, hookFeesAmount)
		inputAmount.Add(inputAmount, curatorFeeAmount)

		var actualOutputAmount uint256.Int
		if amountSpecified.Gt(outputAmount) {
			actualOutputAmount.Set(outputAmount)
		}

		result.DeltaSpecific = actualOutputAmount.ToBig()
		result.DeltaUnSpecific = inputAmount.ToBig()

		hookHandleSwapOutputAmount = outputAmount.Clone()

		hookHandleSwapInputAmount.Sub(hookHandleSwapInputAmount, hookFeesAmount)
		hookHandleSwapInputAmount.Sub(hookHandleSwapInputAmount, curatorFeeAmount)

		if useAmAmmFee {
			hookHandleSwapInputAmount.Sub(hookHandleSwapInputAmount, swapFeeAmount)
		}
	}

	err = h.hookHandleSwap(params.ZeroForOne, hookHandleSwapInputAmount, hookHandleSwapOutputAmount, shouldSurge)
	if err != nil {
		return nil, err
	}

	// rebalanceOrderDeadline := h.rebalanceOrderDeadline
	// if shouldSurge {
	// 	rebalanceOrderDeadline = uint256.NewInt(0)
	// }

	// if h.hookParams.RebalanceThreshold != 0 &&
	// 	(shouldSurge || blockTimestamp > uint32(rebalanceOrderDeadline.Uint64()) && !rebalanceOrderDeadline.IsZero()) {
	// 	if shouldSurge {
	// 		h.rebalanceOrderDeadline.Clear()
	// 	}

	// 	h.rebalance()
	// }

	h.hooklet.AfterSwap(nil)

	return &result, nil
}

func (h *Hook) rebalance() error {
	// should implement to avoid unexpected revert
	return nil
}

func (h *Hook) hookHandleSwap(zeroForOne bool, inputAmount, outputAmount *uint256.Int, shouldSurge bool) error {
	if !inputAmount.IsZero() {
		if zeroForOne {
			h.bunniState.RawBalance0.Add(h.bunniState.RawBalance0, inputAmount)
		} else {
			h.bunniState.RawBalance1.Add(h.bunniState.RawBalance1, inputAmount)
		}
	}

	if !outputAmount.IsZero() {
		outputRawBalance, vaultIndex := h.bunniState.RawBalance0, 0
		if zeroForOne {
			outputRawBalance, vaultIndex = h.bunniState.RawBalance1, 1
		}

		outputVault := h.vaults[vaultIndex]

		if !valueobject.IsZeroAddress(outputVault.Address) && outputRawBalance.Lt(outputAmount) {
			rawBalanceChange := i256.SafeToInt256(new(uint256.Int).Sub(outputAmount, outputRawBalance))

			reserveChange, rawBalanceChange, err := h.updateVaultReserveViaClaimTokens(vaultIndex, rawBalanceChange)
			if err != nil {
				return err
			}

			if zeroForOne {
				h.bunniState.Reserve1.Add(h.bunniState.Reserve1, i256.SafeConvertToUInt256(reserveChange))
				h.bunniState.RawBalance1.Add(h.bunniState.RawBalance1, i256.SafeConvertToUInt256(rawBalanceChange))
			} else {
				h.bunniState.Reserve0.Add(h.bunniState.Reserve0, i256.SafeConvertToUInt256(reserveChange))
				h.bunniState.RawBalance0.Add(h.bunniState.RawBalance0, i256.SafeConvertToUInt256(rawBalanceChange))
			}
		}

		if zeroForOne {
			h.bunniState.RawBalance1.Sub(h.bunniState.RawBalance1, outputAmount)
		} else {
			h.bunniState.RawBalance0.Sub(h.bunniState.RawBalance0, outputAmount)
		}
	}

	if !shouldSurge {
		if !valueobject.IsZeroAddress(h.vaults[0].Address) {
			newReserve0, newRawBalance0, err := h.updateRawBalanceIfNeeded(
				0,
				h.bunniState.RawBalance0, h.bunniState.Reserve0,
				h.bunniState.MinRawTokenRatio0, h.bunniState.MaxRawTokenRatio0, h.bunniState.TargetRawTokenRatio0)
			if err != nil {
				return err
			}

			h.bunniState.Reserve0.Set(newReserve0)
			h.bunniState.RawBalance0.Set(newRawBalance0)
		}

		if !valueobject.IsZeroAddress(h.vaults[1].Address) {
			newReserve1, newRawBalance1, err := h.updateRawBalanceIfNeeded(
				1,
				h.bunniState.RawBalance1, h.bunniState.Reserve1,
				h.bunniState.MinRawTokenRatio1, h.bunniState.MaxRawTokenRatio1, h.bunniState.TargetRawTokenRatio1)
			if err != nil {
				return err
			}

			h.bunniState.Reserve1.Set(newReserve1)
			h.bunniState.RawBalance1.Set(newRawBalance1)
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
	reserveInUnderlying, err := getReservesInUnderlying(h.vaults[vaultIndex], reserve)
	if err != nil {
		return nil, nil, err
	}

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

		var reserveChange *int256.Int
		reserveChange, rawBalanceChange, err = h.updateVaultReserveViaClaimTokens(vaultIndex, rawBalanceChange)
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

func getReservesInUnderlying(vault Vault, reserveAmount *uint256.Int) (*uint256.Int, error) {
	if valueobject.IsZeroAddress(vault.Address) {
		return reserveAmount, nil
	}

	return v3Utils.MulDivRoundingUp(reserveAmount, WAD, vault.RedeemRate)
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
		absAmount = u256.Min(u256.Min(absAmount, h.vaults[index].MaxDeposit), h.poolManagerReserves[index])

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
	SqrtPriceX96                     *uint256.Int
	CurrentTick                      int
	ArithmeticMeanTick               int
	ZeroForOne                       bool
	ExactIn                          bool
	AmountSpecified                  *uint256.Int
	SqrtPriceLimitX96                *uint256.Int
	LdfState                         [32]byte
}

func (h *Hook) computeSwap(input BunniComputeSwapInput) (*uint256.Int, int, *uint256.Int, *uint256.Int, error) {
	// Initialize input and output amounts based on initial info
	var inputAmount, outputAmount uint256.Int
	if input.ExactIn {
		inputAmount.Set(input.AmountSpecified)
	} else {
		outputAmount.Set(input.AmountSpecified)
	}

	// Initialize updatedTick to the current tick
	updatedTick := input.CurrentTick

	// Compute updated rounded tick liquidity
	var updatedRoundedTickLiquidity uint256.Int
	updatedRoundedTickLiquidity.Mul(input.TotalLiquidity, input.LiquidityDensityOfRoundedTickX96)
	updatedRoundedTickLiquidity.Rsh(&updatedRoundedTickLiquidity, 96)

	// Bound sqrtPriceLimitX96 by min/max possible values
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

	// Bound sqrtPriceLimit so that we never end up at an invalid rounded tick
	if (input.ZeroForOne && sqrtPriceLimitX96.Cmp(minSqrtPrice) <= 0) ||
		(!input.ZeroForOne && sqrtPriceLimitX96.Cmp(maxSqrtPrice) >= 0) {
		if input.ZeroForOne {
			sqrtPriceLimitX96.AddUint64(minSqrtPrice, 1)
		} else {
			sqrtPriceLimitX96.SubUint64(maxSqrtPrice, 1)
		}
	}

	roundedTick, nextRoundedTick := math.RoundTick(input.CurrentTick, h.tickSpacing)

	var naiveSwapResultSqrtPriceX96, naiveSwapAmountIn, naiveSwapAmountOut *uint256.Int

	// Handle the special case when we don't cross rounded ticks
	if !updatedRoundedTickLiquidity.IsZero() {
		tickNext := lo.Ternary(input.ZeroForOne, roundedTick, nextRoundedTick)

		sqrtPriceNextX96, err := math.GetSqrtPriceAtTick(tickNext)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		// Get sqrt price target
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
			input.ExactIn, input.ZeroForOne, input.SqrtPriceX96,
			&sqrtPriceTargetX96, &updatedRoundedTickLiquidity, input.AmountSpecified, u256.U0)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		// Check if naive swap exhausted the specified amount
		if (input.ExactIn && naiveSwapAmountIn.Eq(input.AmountSpecified)) ||
			(!input.ExactIn && naiveSwapAmountOut.Eq(input.AmountSpecified)) {

			// Compute the updated tick
			if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceNextX96) {
				updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
			} else if !naiveSwapResultSqrtPriceX96.Eq(input.SqrtPriceX96) {
				updatedTick, err = math.GetTickAtSqrtPrice(naiveSwapResultSqrtPriceX96)
				if err != nil {
					return nil, 0, nil, nil, err
				}
			}

			// naiveSwapAmountOut should be at most the corresponding active balance
			currentBalance := lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0)
			if naiveSwapAmountOut.Gt(currentBalance) {
				naiveSwapAmountOut.Set(currentBalance)
			}

			// Early return
			return naiveSwapResultSqrtPriceX96, updatedTick, naiveSwapAmountIn, naiveSwapAmountOut, nil
		}
	}

	// Swap crosses rounded tick - need to use LDF to compute the swap
	var inverseCumulativeAmountFnInput uint256.Int
	if input.ExactIn {
		// Exact input swap
		inverseCumulativeAmountFnInput.Set(lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance0, input.CurrentActiveBalance1))
		inverseCumulativeAmountFnInput.Add(&inverseCumulativeAmountFnInput, &inputAmount)
	} else {
		// Exact output swap
		inverseCumulativeAmountFnInput.Set(lo.Ternary(input.ZeroForOne, input.CurrentActiveBalance1, input.CurrentActiveBalance0))
		inverseCumulativeAmountFnInput.Sub(&inverseCumulativeAmountFnInput, &outputAmount)
	}

	// Call LDF computeSwap
	success, updatedRoundedTick, cumulativeAmount0, cumulativeAmount1, swapLiquidity, err := h.ldf.ComputeSwap(
		&inverseCumulativeAmountFnInput,
		input.TotalLiquidity,
		input.ZeroForOne,
		input.ExactIn,
		input.ArithmeticMeanTick,
		input.CurrentTick,
		h.bunniState.LdfParams,
		h.ldfState,
	)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	if success {
		if (input.ZeroForOne && updatedRoundedTick >= roundedTick) ||
			(!input.ZeroForOne && updatedRoundedTick <= roundedTick) {

			if updatedRoundedTickLiquidity.IsZero() {
				return input.SqrtPriceX96, input.CurrentTick, uint256.NewInt(0), uint256.NewInt(0), nil
			}

			tickNext := lo.Ternary(input.ZeroForOne, roundedTick, nextRoundedTick)

			sqrtPriceNextX96, err := math.GetSqrtPriceAtTick(tickNext)
			if err != nil {
				return nil, 0, nil, nil, err
			}

			if naiveSwapResultSqrtPriceX96.Eq(sqrtPriceNextX96) {
				updatedTick = lo.Ternary(input.ZeroForOne, tickNext-1, tickNext)
			} else if !naiveSwapResultSqrtPriceX96.Eq(input.SqrtPriceX96) {
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

		if input.ZeroForOne && sqrtPriceLimitX96.Cmp(startSqrtPriceX96) < 0 ||
			(!input.ZeroForOne && sqrtPriceLimitX96.Cmp(startSqrtPriceX96) > 0) {

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

					// Compute input and output token amounts
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

	// The sqrt price limit has been reached
	var updatedSqrtPriceX96 uint256.Int
	updatedSqrtPriceX96.Set(&sqrtPriceLimitX96)

	if sqrtPriceLimitX96.Eq(input.SqrtPriceX96) {
		updatedTick = input.CurrentTick
	} else {
		updatedTick, err = math.GetTickAtSqrtPrice(&sqrtPriceLimitX96)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}

	// Query LDF for updated balances
	totalDensity0X96, totalDensity1X96, _, _, _, _, err := h.queryLDF(&updatedSqrtPriceX96, updatedTick,
		input.ArithmeticMeanTick, input.LdfState, u256.U0, u256.U0, ZERO)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	var updatedActiveBalance0, updatedActiveBalance1 uint256.Int
	updatedActiveBalance0.MulDivOverflow(totalDensity0X96, input.TotalLiquidity, SWAP_FEE_BASE)
	updatedActiveBalance1.MulDivOverflow(totalDensity1X96, input.TotalLiquidity, SWAP_FEE_BASE)

	// Compute final input and output amounts
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
) (
	*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, [32]byte, bool, error) {

	roundedTick, nextRoundedTick := math.RoundTick(h.slot0.Tick, h.tickSpacing)

	roundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}
	nextRoundedTickSqrtRatio, err := math.GetSqrtPriceAtTick(nextRoundedTick)
	if err != nil {
		return nil, nil, nil, nil, ldfState, false, err
	}

	liquidityDensityOfRoundedTickX96, density0RightOfRoundedTickX96, density1LeftOfRoundedTickX96,
		newLdfState, shouldSurge, err := h.ldf.Query(roundedTick, arithmeticMeanTick, tick, h.bunniState.LdfParams, h.ldfState)
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

		useLiquidityEstimate0 := (totalLiquidityEstimate0.Lt(totalLiquidityEstimate1) || totalDensity1X96.IsZero()) && !totalDensity0X96.IsZero()

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
	sqrtPriceX96, sqrtPriceAX96, sqrtPriceBX96, liquidity *uint256.Int,
	roundUp bool,
) (*uint256.Int, *uint256.Int, error) {
	var amount0, amount1 uint256.Int

	// Ensure sqrtPriceAX96 <= sqrtPriceBX96
	if sqrtPriceAX96.Gt(sqrtPriceBX96) {
		sqrtPriceAX96, sqrtPriceBX96 = sqrtPriceBX96, sqrtPriceAX96
	}

	if sqrtPriceX96.Cmp(sqrtPriceAX96) <= 0 {
		// Current price is at or below the lower bound
		amount0Delta, err := math.GetAmount0Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount0.Set(amount0Delta)
	} else if sqrtPriceX96.Lt(sqrtPriceBX96) {
		// Current price is within the range
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
		// Current price is at or above the upper bound
		amount1Delta, err := math.GetAmount1Delta(sqrtPriceAX96, sqrtPriceBX96, liquidity, roundUp)
		if err != nil {
			return nil, nil, err
		}
		amount1.Set(amount1Delta)
	}

	return &amount0, &amount1, nil
}

func (h *Hook) getTwap(
	blockTimestamp uint32,
) (int64, error) {
	tickCumulatives, err := h.observation.ObserveDouble(h.state.IntermediateObservation, blockTimestamp,
		[]uint32{h.bunniState.TwapSecondsAgo, 0}, h.slot0.Tick, h.state.Index, h.state.Cardinality)
	if err != nil {
		return 0, err
	}

	tickCumulativesDelta := tickCumulatives[0] - tickCumulatives[1]

	return tickCumulativesDelta / int64(h.bunniState.TwapSecondsAgo), nil
}

func (h *Hook) updateOracle(blockTimestamp uint32) {
	h.state.IntermediateObservation, h.state.Index, h.state.Cardinality =
		h.observation.Write(h.state.IntermediateObservation,
			h.state.Index, blockTimestamp, h.slot0.Tick, h.state.Cardinality,
			h.state.CardinalityNext, h.hookParams.OracleMinInterval)
}

func computeSurgeFee(blockTimestamp, lastSurgeTimestamp uint32, surgeFeeHalfLife *uint256.Int) (*uint256.Int, error) {
	timeSinceLastSurge := uint256.NewInt(uint64(blockTimestamp - lastSurgeTimestamp))

	fee, _ := new(uint256.Int).MulDivOverflow(timeSinceLastSurge, LN2_WAD, surgeFeeHalfLife)

	var err error
	fee, err = math.ExpWad(i256.SafeToInt256(fee))
	if err != nil {
		return nil, err
	}

	fee, err = math.MulWadUp(SWAP_FEE_BASE, fee)
	if err != nil {
		return nil, err
	}

	return fee, nil
}

func computeDynamicSwapFee(blockTimestamp uint32, postSwapSqrtPriceX96 *uint256.Int,
	arithmeticMeanTick int, lastSurgeTimestamp uint32,
	feeMin, feeMax, feeQuadraticMultiplier, surgeFeeHalfLife *uint256.Int) (*uint256.Int, error) {

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
	if !valueobject.IsZeroAddress(h.vaults[0].Address) || !valueobject.IsZeroAddress(h.vaults[1].Address) {
		rescaleFactor0 := 18 + h.bunniState.Vault0Decimals - h.bunniState.Currency0Decimals
		rescaleFactor1 := 18 + h.bunniState.Vault1Decimals - h.bunniState.Currency1Decimals

		var sharePrice0 *uint256.Int
		if !h.bunniState.Reserve0.IsZero() {
			sharePrice0, err = v3Utils.MulDivRoundingUp(h.reserveBalances[0], u256.TenPow(rescaleFactor0), h.bunniState.Reserve0)
			if err != nil {
				return false, err
			}
		}

		var sharePrice1 *uint256.Int
		if !h.bunniState.Reserve1.IsZero() {
			sharePrice1, err = v3Utils.MulDivRoundingUp(h.reserveBalances[1], u256.TenPow(rescaleFactor1), h.bunniState.Reserve1)
			if err != nil {
				return false, err
			}
		}

		shouldSurge = h.prevSharePrices.Initialized &&
			(math.Dist(sharePrice0, h.prevSharePrices.SharedPrice0).
				Gt(new(uint256.Int).Div(h.prevSharePrices.SharedPrice0, h.hookParams.VaultSurgeThreshold0)) ||
				math.Dist(sharePrice1, h.prevSharePrices.SharedPrice1).
					Gt(new(uint256.Int).Div(h.prevSharePrices.SharedPrice1, h.hookParams.VaultSurgeThreshold1)))

		if !h.prevSharePrices.Initialized || !sharePrice0.Eq(h.prevSharePrices.SharedPrice0) ||
			!sharePrice1.Eq(h.prevSharePrices.SharedPrice1) {
			h.prevSharePrices.Initialized = true
			h.prevSharePrices.SharedPrice0.Set(sharePrice0)
			h.prevSharePrices.SharedPrice1.Set(sharePrice1)
		}
	}

	return shouldSurge, nil
}
