package stablestable

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook          `json:"-"`
	uniswapv3.ExtraTickU256 `json:"-"`

	K                     uint64 `json:"k"`
	LogK                  uint64 `json:"lk"`
	OptimalFeeE6          uint64 `json:"o"`
	TargetMultiplier      uint64 `json:"tm"`
	ReferenceSqrtPriceX96 string `json:"rsp"`

	DecayingFeeE12  uint64 `json:"df"`
	SqrtAmmPriceX96 string `json:"sap"` // decimal string; 0 means "force fresh read"
	BlockNumber     uint64 `json:"bn"`

	TrackedBlock uint64 `json:"tb"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4StableStable},
	}
	_ = param.HookExtra.Unmarshal(hook)
	if param.Pool != nil && param.Pool.Extra != "" {
		_ = json.Unmarshal([]byte(param.Pool.Extra), &hook.ExtraTickU256)
	}
	return hook
}, HookAddresses...)

type feeConfigRPC struct {
	K                     *big.Int
	LogK                  *big.Int
	OptimalFeeE6          *big.Int
	TargetMultiplier      uint8
	ReferenceSqrtPriceX96 *big.Int
}

type feeStateRPC struct {
	DecayingFeeE12  *big.Int
	SqrtAmmPriceX96 *big.Int
	BlockNumber     *big.Int
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	var (
		cfg   feeConfigRPC
		state feeStateRPC
	)

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	poolId := eth.StringToBytes32(param.Pool.Address)
	req.AddCall(&ethrpc.Call{
		ABI:    stableStableHookABI,
		Target: param.HookAddress.Hex(),
		Method: "feeConfig",
		Params: []any{poolId},
	}, []any{&cfg})
	req.AddCall(&ethrpc.Call{
		ABI:    stableStableHookABI,
		Target: param.HookAddress.Hex(),
		Method: "feeState",
		Params: []any{poolId},
	}, []any{&state})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	h.K = cfg.K.Uint64()
	h.LogK = cfg.LogK.Uint64()
	h.OptimalFeeE6 = cfg.OptimalFeeE6.Uint64()
	h.TargetMultiplier = uint64(cfg.TargetMultiplier)
	h.ReferenceSqrtPriceX96 = cfg.ReferenceSqrtPriceX96.String()
	h.DecayingFeeE12 = state.DecayingFeeE12.Uint64()
	h.SqrtAmmPriceX96 = state.SqrtAmmPriceX96.String()
	h.BlockNumber = state.BlockNumber.Uint64()
	if param.BlockNumber != nil {
		h.TrackedBlock = param.BlockNumber.Uint64()
	}

	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.OptimalFeeE6 == 0 || h.OptimalFeeE6 > MaxOptimalFeeE6 {
		return nil, ErrInvalidFeeConfig
	}
	if h.SqrtPriceX96 == nil || h.SqrtPriceX96.IsZero() {
		return nil, ErrInvalidFeeConfig
	}

	sqrtRefX96, err := uint256.FromDecimal(h.ReferenceSqrtPriceX96)
	if err != nil || sqrtRefX96.IsZero() {
		return nil, ErrInvalidFeeConfig
	}

	cachedSqrtPrev, err := uint256.FromDecimal(h.SqrtAmmPriceX96)
	if err != nil || cachedSqrtPrev == nil {
		cachedSqrtPrev = new(uint256.Int)
	}

	currentSqrt := new(uint256.Int).Set(h.SqrtPriceX96)

	isNewBlock := h.TrackedBlock != h.BlockNumber || cachedSqrtPrev.IsZero()
	var sqrtAmmPriceX96 *uint256.Int
	if isNewBlock {
		sqrtAmmPriceX96 = currentSqrt
	} else {
		sqrtAmmPriceX96 = cachedSqrtPrev
	}

	priceRatioX96 := CalculatePriceRatioX96(sqrtAmmPriceX96, sqrtRefX96)
	closeBoundaryMagE12, isOutside := CalculateCloseBoundaryFee(priceRatioX96, h.OptimalFeeE6)

	userSellsZeroForOne := params.ZeroForOne
	ammPriceBelowRP := sqrtAmmPriceX96.Cmp(sqrtRefX96) < 0

	var (
		lpFeeE12                 *uint256.Int
		nextDecayingFeeE12Stored uint64
	)

	if !isOutside {
		// Inside optimal range: charge the fee that puts the pre-impact price
		// exactly on the corresponding boundary.
		fee, err := CalculateInsideOptimalRangeFee(priceRatioX96, h.OptimalFeeE6, ammPriceBelowRP, userSellsZeroForOne)
		if err != nil {
			return nil, err
		}
		lpFeeE12 = fee
		nextDecayingFeeE12Stored = undefinedDecayingFeeE12.Uint64()
	} else {
		// Outside optimal range: compute the decaying fee.
		var decayingFeeE12 *uint256.Int
		if isNewBlock {
			farBoundaryFeeE12 := CalculateFarBoundaryFee(priceRatioX96, h.OptimalFeeE6)
			df, err := h.calculateDecayingFee(
				sqrtAmmPriceX96, sqrtRefX96,
				closeBoundaryMagE12, farBoundaryFeeE12,
				ammPriceBelowRP,
				cachedSqrtPrev,
			)
			if err != nil {
				return nil, err
			}
			decayingFeeE12 = df
		} else {
			decayingFeeE12 = uint256.NewInt(h.DecayingFeeE12)
		}
		nextDecayingFeeE12Stored = decayingFeeE12.Uint64()

		// Direction selector: if the swap pushes price further from the
		// reference, charge zero. Otherwise charge the decaying fee.
		if ammPriceBelowRP == userSellsZeroForOne {
			lpFeeE12 = new(uint256.Int)
		} else {
			lpFeeE12 = decayingFeeE12
		}
	}

	swapFeeE6 := new(uint256.Int).Div(lpFeeE12, oneE6).Uint64()
	if swapFeeE6 > uint64(uniswapv4.FeeMax) {
		swapFeeE6 = uint64(uniswapv4.FeeMax)
	}

	var info *stableStableSwapInfo
	if isNewBlock {
		info = &stableStableSwapInfo{
			sqrtAmmPriceX96: sqrtAmmPriceX96.Dec(),
			decayingFeeE12:  nextDecayingFeeE12Stored,
			blockNumber:     h.TrackedBlock,
		}
	}

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   bignumber.ZeroBI,
		DeltaUnspecified: bignumber.ZeroBI,
		SwapFee:          uniswapv4.FeeAmount(swapFeeE6),
		Gas:              gasBeforeSwap,
		SwapInfo:         info,
	}, nil
}

type stableStableSwapInfo struct {
	sqrtAmmPriceX96 string
	decayingFeeE12  uint64
	blockNumber     uint64
}

func (h *Hook) UpdateBalance(info any) {
	si, ok := info.(*stableStableSwapInfo)
	if !ok || si == nil {
		return
	}
	h.SqrtAmmPriceX96 = si.sqrtAmmPriceX96
	h.DecayingFeeE12 = si.decayingFeeE12
	h.BlockNumber = si.blockNumber
}

// calculateDecayingFee mirrors StableStableHook._calculateDecayingFee.
func (h *Hook) calculateDecayingFee(
	sqrtAmmPriceX96, sqrtRefX96, closeBoundaryFeeE12, farBoundaryFeeE12 *uint256.Int,
	ammPriceBelowRP bool,
	previousSqrtAmmPriceX96 *uint256.Int,
) (*uint256.Int, error) {
	previousDecayingFeeE12 := uint256.NewInt(h.DecayingFeeE12)
	previousBlockNumber := h.BlockNumber

	var decayStartFeeE12 *uint256.Int
	previousAmmBelowRP := previousSqrtAmmPriceX96.Cmp(sqrtRefX96) < 0

	switch {
	case previousDecayingFeeE12.Cmp(undefinedDecayingFeeE12) == 0 || previousAmmBelowRP != ammPriceBelowRP:
		// Just left the optimal range, or jumped across the reference: start
		// from the far boundary.
		decayStartFeeE12 = farBoundaryFeeE12
	case ammPriceBelowRP == (sqrtAmmPriceX96.Cmp(previousSqrtAmmPriceX96) < 0):
		// Price moved further from the reference. Adjust previous fee so the
		// pre-impact price is preserved, then decay from there.
		movementRatio := CalculatePriceRatioX96(sqrtAmmPriceX96, previousSqrtAmmPriceX96)
		decayStartFeeE12 = AdjustPreviousFeeForPriceMovement(movementRatio, previousDecayingFeeE12)
	case previousDecayingFeeE12.Cmp(farBoundaryFeeE12) > 0:
		// Price moved toward the reference and previousFee now exceeds the
		// new far boundary fee — cap at the new far boundary.
		decayStartFeeE12 = farBoundaryFeeE12
	default:
		decayStartFeeE12 = previousDecayingFeeE12
	}

	// targetFee = farBoundaryFee - closeBoundaryFee * targetMultiplier / 100
	targetSubtrahend := new(uint256.Int).Mul(closeBoundaryFeeE12, uint256.NewInt(h.TargetMultiplier))
	targetSubtrahend.Div(targetSubtrahend, maxTargetMultiplierU)
	if targetSubtrahend.Gt(farBoundaryFeeE12) {
		return nil, ErrInvalidFeeConfig
	}
	targetFee := new(uint256.Int).Sub(farBoundaryFeeE12, targetSubtrahend)

	var blocksPassed uint64
	if h.TrackedBlock > previousBlockNumber {
		blocksPassed = h.TrackedBlock - previousBlockNumber
	}

	return CalculateDecayingFee(targetFee, decayStartFeeE12, h.K, h.LogK, blocksPassed)
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	if h.SqrtPriceX96 != nil {
		cloned.SqrtPriceX96 = new(uint256.Int).Set(h.SqrtPriceX96)
	}
	return &cloned
}

var _ uniswapv4.Hook = (*Hook)(nil)
