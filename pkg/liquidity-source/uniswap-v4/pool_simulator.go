package uniswapv4

import (
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/few"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	defaultGas = uniswapv3.Gas{BaseGas: 75000, CrossInitTickGas: 21000}
)

type PoolSimulator struct {
	*uniswapv3.PoolSimulator
	staticExtra StaticExtra
	hook        Hook
	chainID     valueobject.ChainID
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	hook, ok := GetHook(staticExtra.HooksAddress, &HookParam{
		Cfg:       &Config{ChainID: int(chainID)},
		Pool:      &entityPool,
		HookExtra: extra.HookExtra,
	})
	if !ok && HasSwapPermissions(staticExtra.HooksAddress) {
		return nil, shared.ErrUnsupportedHook
	}

	if shared.IsDynamicFee(uint32(entityPool.SwapFee)) {
		entityPool.SwapFee = 0
	}

	v3PoolSimulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, chainID, extra.ExtraTickU256)
	if err != nil {
		return nil, errors.WithMessage(pool.ErrUnsupported, err.Error())
	}
	if entityPool.Tokens[0].Address > entityPool.Tokens[1].Address {
		// restore original order after V3Pool constructor forced sorting
		v3Pool := v3PoolSimulator.V3Pool
		v3Pool.Token0, v3Pool.Token1 = v3Pool.Token1, v3Pool.Token0
	}
	v3PoolSimulator.Gas = defaultGas
	return &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		staticExtra:   staticExtra,
		hook:          hook,
		chainID:       chainID,
	}, nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	tokenIndex := p.GetTokenIndex(address)
	var canWrapToFew bool
	var fewInfo few.TokenInfo

	if tokenIndex == -1 {
		fewInfo, canWrapToFew = few.CanWrapToFew(p.chainID, address)
		if !canWrapToFew {
			return nil
		}
		fewTokenIndex := p.GetTokenIndex(fewInfo.FewTokenAddress)
		if fewTokenIndex == -1 {
			return nil
		}
	}

	result := map[string]struct{}{}
	for _, token := range p.Info.Tokens {
		if (tokenIndex >= 0 && token != address) || (tokenIndex == -1 && token != fewInfo.FewTokenAddress) {
			result[token] = struct{}{}

			fewInfo, isFewToken := few.IsFewToken(p.chainID, token)
			if isFewToken && fewInfo.UnwrapTokenAddress != address {
				result[fewInfo.UnwrapTokenAddress] = struct{}{}
			}
		}
	}

	return slices.Collect(maps.Keys(result))
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
}

func (p *PoolSimulator) GetTokens() []string {
	result := make(map[string]struct{})

	for _, token := range p.Info.Tokens {
		result[token] = struct{}{}
		if fewInfo, isFewToken := few.IsFewToken(p.chainID, token); isFewToken {
			result[fewInfo.UnwrapTokenAddress] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(result))
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (result *pool.CalcAmountOutResult, err error) {
	originalTokenIn := param.TokenAmountIn.Token
	originalTokenOut := param.TokenOut
	wrapFewAdditionalGas := int64(0)

	defer func() {
		if result.TokenAmountOut != nil {
			result.TokenAmountOut.Token = originalTokenOut
		}
		if result.RemainingTokenAmountIn != nil {
			result.RemainingTokenAmountIn.Token = originalTokenIn
		}

		result.Gas += wrapFewAdditionalGas
	}()

	// Wrap/unwrap FEW token if needed.
	// Assume that if the token is not in the pool, it is a FEW token.
	if p.GetTokenIndex(param.TokenAmountIn.Token) == -1 {
		fewInfo, canWrapToFew := few.CanWrapToFew(p.chainID, param.TokenAmountIn.Token)
		if canWrapToFew {
			param.TokenAmountIn.Token = fewInfo.FewTokenAddress
			wrapFewAdditionalGas += p.Gas.BaseGas
		}
	}
	if p.GetTokenIndex(param.TokenOut) == -1 {
		fewInfo, canUnwrapToFew := few.CanWrapToFew(p.chainID, param.TokenOut)
		if canUnwrapToFew {
			param.TokenOut = fewInfo.FewTokenAddress
			wrapFewAdditionalGas += p.Gas.BaseGas
		}
	}

	poolSim := p.PoolSimulator
	if p.hook == nil {
		result, err = poolSim.CalcAmountOut(param)
		return
	}

	tokenIn := param.TokenAmountIn.Token

	beforeSwapHookParams := &BeforeSwapHookParams{
		ExactIn:         true,
		ZeroForOne:      p.Pool.GetTokenIndex(tokenIn) == 0,
		AmountSpecified: param.TokenAmountIn.Amount,
	}
	swapHookResult, err := p.hook.BeforeSwap(beforeSwapHookParams)
	if err != nil {
		return nil, err
	}

	// beforeSwap -> amountToSwap += hookDeltaSpecified;
	// for the case of calcAmountOut (exactIn) means amountToSwap(amountIn) supposed to be Negative, then turn to be TokenAmountIn.Sub
	// fot the case of calcAmountIn (exactOut) means amountToSwap(amountOut) supposed to be Positive, then turn to be TokenAmountOut.Add
	var amountIn *big.Int
	if swapHookResult != nil && swapHookResult.DeltaSpecific != nil {
		amountIn = new(big.Int).Sub(param.TokenAmountIn.Amount, swapHookResult.DeltaSpecific)
	} else {
		amountIn = param.TokenAmountIn.Amount
	}

	if swapHookResult.SwapFee >= constants.FeeMax {
		return nil, errors.New("swap disabled")
	} else if swapHookResult.SwapFee > 0 && swapHookResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = swapHookResult.SwapFee
		poolSim = &cloned
	}

	result, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: param.TokenOut,
	})
	if err != nil {
		return nil, err
	}
	hookFee := p.hook.AfterSwap(&AfterSwapHookParams{
		BeforeSwapHookParams: beforeSwapHookParams,
		AmountIn:             amountIn,
		AmountOut:            result.TokenAmountOut.Amount,
	})

	// afterSwap -> swapDelta = swapDelta - hookDelta;
	// for the case of calcAmountOut (exactIn), amountOut supposed to be Positive, then turn to be TokenAmountOut.Sub
	// for the case of calcAmountIn (exactOut), amountIn supposed to be Negative, then turn to be TokenAmountIn.Add
	if swapHookResult != nil && swapHookResult.DeltaUnSpecific != nil {
		result.TokenAmountOut.Amount.Sub(result.TokenAmountOut.Amount, swapHookResult.DeltaUnSpecific)
	}
	if hookFee != nil {
		result.TokenAmountOut.Amount.Sub(result.TokenAmountOut.Amount, hookFee)
	}
	return result, err
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	if cloned.hook != nil {
		cloned.hook = p.hook.CloneState()
	}
	return &cloned
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	tokenInAfterWrapFew := tokenIn
	tokenOutBeforeUnwrapFew := tokenOut

	var wrapFewMetadata few.WrapFewMetadata
	tokenInIndex := p.GetTokenIndex(tokenIn)
	if tokenInIndex == -1 {
		fewInfo, canWrapToFew := few.CanWrapToFew(p.chainID, tokenIn)
		if canWrapToFew {
			wrapFewMetadata.ShouldWrapFew = true
			tokenInAfterWrapFew = fewInfo.FewTokenAddress

			tokenIn := fewInfo.UnwrapTokenAddress
			if fewInfo.IsNative {
				tokenIn = NativeTokenAddress.Hex()
			}
			wrapFewMetadata.WrapFewInfo = few.WrapFewInfo{
				TokenIn:     tokenIn,
				TokenOut:    fewInfo.FewTokenAddress,
				HookAddress: fewInfo.HookAddress,
				PoolAddress: fewInfo.PoolAddress,
				TickSpacing: fewInfo.TickSpacing,
				Fee:         fewInfo.Fee,
			}
		}
	}

	tokenOutIndex := p.GetTokenIndex(tokenOut)
	if tokenOutIndex == -1 {
		fewInfo, canUnwrapToFew := few.CanWrapToFew(p.chainID, tokenOut)
		if canUnwrapToFew {
			wrapFewMetadata.ShouldUnwrapFew = true
			tokenOutBeforeUnwrapFew = fewInfo.FewTokenAddress

			tokenOut := fewInfo.UnwrapTokenAddress
			if fewInfo.IsNative {
				tokenOut = NativeTokenAddress.Hex()
			}
			wrapFewMetadata.UnwrapFewInfo = few.WrapFewInfo{
				TokenIn:     fewInfo.FewTokenAddress,
				TokenOut:    tokenOut,
				HookAddress: fewInfo.HookAddress,
				PoolAddress: fewInfo.PoolAddress,
				TickSpacing: fewInfo.TickSpacing,
				Fee:         fewInfo.Fee,
			}
		}
	}

	tokenInAddress, tokenOutAddress := NativeTokenAddress, NativeTokenAddress
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenInAfterWrapFew)] {
		tokenInAddress = common.HexToAddress(tokenInAfterWrapFew)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOutBeforeUnwrapFew)] {
		tokenOutAddress = common.HexToAddress(tokenOutBeforeUnwrapFew)
	}
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(tokenInAfterWrapFew == p.Info.Tokens[0], &priceLimit)

	return PoolMetaInfo{
		Router:          p.staticExtra.UniversalRouterAddress,
		Permit2Addr:     p.staticExtra.Permit2Address,
		TokenIn:         tokenInAddress,
		TokenOut:        tokenOutAddress,
		Fee:             p.staticExtra.Fee,
		TickSpacing:     p.staticExtra.TickSpacing,
		HookAddress:     p.staticExtra.HooksAddress,
		HookData:        []byte{},
		PriceLimit:      &priceLimit,
		WrapFewMetadata: wrapFewMetadata,
	}
}
