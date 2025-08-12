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
	staticExtra   StaticExtra
	hook          Hook
	chainID       valueobject.ChainID
	tokenWrappers []ITokenWrapper
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
		tokenWrappers: []ITokenWrapper{few.NewTokenWrapper()},
	}, nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	tokenIndex := p.GetTokenIndex(address)
	var wrapTokens = make(map[string]struct{})

	if tokenIndex == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, address)
			if !canWrap {
				continue
			}

			wrapTokenIndex := p.GetTokenIndex(metadata.GetWrapToken())
			if wrapTokenIndex == -1 {
				continue
			}

			wrapTokens[metadata.GetWrapToken()] = struct{}{}
		}
	}

	result := map[string]struct{}{}
	for _, token := range p.Info.Tokens {
		_, isWrapToken := wrapTokens[token]
		if (tokenIndex >= 0 && token != address) || (tokenIndex == -1 && !isWrapToken) {
			result[token] = struct{}{}

			for _, wrapper := range p.tokenWrappers {
				metadata, canUnwrap := wrapper.IsWrapped(p.chainID, token)
				if canUnwrap && metadata.GetUnwrapToken() != address {
					result[metadata.GetUnwrapToken()] = struct{}{}
				}
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
		for _, wrapper := range p.tokenWrappers {
			if metadata, isWrapped := wrapper.IsWrapped(p.chainID, token); isWrapped {
				result[metadata.GetUnwrapToken()] = struct{}{}
			}
		}
	}

	return slices.Collect(maps.Keys(result))
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (result *pool.CalcAmountOutResult, err error) {
	originalTokenIn := param.TokenAmountIn.Token
	originalTokenOut := param.TokenOut
	wrapAdditionalGas := int64(0)

	defer func() {
		if result == nil {
			return
		}
		if result.TokenAmountOut != nil {
			result.TokenAmountOut.Token = originalTokenOut
		}
		if result.RemainingTokenAmountIn != nil {
			result.RemainingTokenAmountIn.Token = originalTokenIn
		}

		result.Gas += wrapAdditionalGas
	}()

	// Wrap/unwrap tokens if needed.
	if p.GetTokenIndex(param.TokenAmountIn.Token) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, param.TokenAmountIn.Token)
			if canWrap {
				param.TokenAmountIn.Token = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}
	if p.GetTokenIndex(param.TokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, param.TokenOut)
			if canUnwrap {
				param.TokenOut = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
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
	if err != nil || swapHookResult == nil {
		return nil, err
	}

	// beforeSwap -> amountToSwap += hookDeltaSpecified;
	// for the case of calcAmountOut (exactIn) means amountToSwap(amountIn) supposed to be Negative, then turn to be TokenAmountIn.Sub
	// fot the case of calcAmountIn (exactOut) means amountToSwap(amountOut) supposed to be Positive, then turn to be TokenAmountOut.Add
	var amountIn *big.Int
	if swapHookResult.DeltaSpecific != nil {
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
	if swapHookResult.DeltaUnSpecific != nil {
		result.TokenAmountOut.Amount.Sub(result.TokenAmountOut.Amount, swapHookResult.DeltaUnSpecific)
	}
	if hookFee != nil {
		result.TokenAmountOut.Amount.Sub(result.TokenAmountOut.Amount, hookFee)
	}
	return result, err
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (result *pool.CalcAmountInResult, err error) {
	originalTokenOut := param.TokenAmountOut.Token
	originalTokenIn := param.TokenIn
	wrapAdditionalGas := int64(0)
	defer func() {
		if result == nil {
			return
		}
		if result.TokenAmountIn != nil {
			result.TokenAmountIn.Token = originalTokenIn
		}

		if result.RemainingTokenAmountOut != nil {
			result.RemainingTokenAmountOut.Token = originalTokenOut
		}

		result.Gas += wrapAdditionalGas
	}()

	// Wrap/unwrap tokens if needed.
	if p.GetTokenIndex(param.TokenAmountOut.Token) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, param.TokenAmountOut.Token)
			if canWrap {
				param.TokenAmountOut.Token = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}

	if p.GetTokenIndex(param.TokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, param.TokenIn)
			if canUnwrap {
				param.TokenIn = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}

	poolSim := p.PoolSimulator
	if p.hook == nil {
		result, err = poolSim.CalcAmountIn(param)
		return
	}

	tokenOut := param.TokenAmountOut.Token
	beforeSwapHookParams := &BeforeSwapHookParams{
		ExactIn:         false,
		ZeroForOne:      p.Pool.GetTokenIndex(tokenOut) == 1,
		AmountSpecified: param.TokenAmountOut.Amount,
	}

	swapHookResult, err := p.hook.BeforeSwap(beforeSwapHookParams)
	if err != nil || swapHookResult == nil {
		return nil, err
	}

	var amountOut *big.Int

	if swapHookResult.DeltaSpecific != nil {
		amountOut = new(big.Int).Add(param.TokenAmountOut.Amount, swapHookResult.DeltaSpecific)
	} else {
		amountOut = param.TokenAmountOut.Amount
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

	result, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		TokenIn: param.TokenIn,
	})
	if err != nil {
		return nil, err
	}

	hookFee := p.hook.AfterSwap(&AfterSwapHookParams{
		BeforeSwapHookParams: beforeSwapHookParams,
		AmountIn:             result.TokenAmountIn.Amount,
		AmountOut:            amountOut,
	})

	if swapHookResult.DeltaUnSpecific != nil {
		result.TokenAmountIn.Amount.Add(result.TokenAmountIn.Amount, swapHookResult.DeltaUnSpecific)
	}
	if hookFee != nil {
		result.TokenAmountIn.Amount.Add(result.TokenAmountIn.Amount, hookFee)
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
	tokenInAfterWrap := tokenIn
	tokenOutBeforeUnwrap := tokenOut

	var wrapMetadata TokenWrapMetadata

	tokenInIndex := p.GetTokenIndex(tokenIn)
	if tokenInIndex == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, tokenIn)
			if canWrap {
				wrapMetadata.ShouldWrap = true
				tokenInAfterWrap = metadata.GetWrapToken()
				tokenIn := tokenIn
				if metadata.IsUnwrapNative() {
					tokenIn = NativeTokenAddress.Hex()
				}

				wrapMetadata.WrapInfo = WrapInfo{
					TokenIn:     tokenIn,
					TokenOut:    metadata.GetWrapToken(),
					HookAddress: metadata.GetHook(),
					PoolAddress: metadata.GetPool(),
					TickSpacing: metadata.GetTickSpacing(),
					Fee:         metadata.GetFee(),
					HookData:    metadata.GetHookData(),
				}
				break
			}
		}
	}

	tokenOutIndex := p.GetTokenIndex(tokenOut)
	if tokenOutIndex == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, tokenOut)
			if canUnwrap {
				wrapMetadata.ShouldUnwrap = true
				tokenOutBeforeUnwrap = metadata.GetWrapToken()
				tokenOut := tokenOut
				if metadata.IsUnwrapNative() {
					tokenOut = NativeTokenAddress.Hex()
				}

				wrapMetadata.UnwrapInfo = WrapInfo{
					TokenIn:     metadata.GetWrapToken(),
					TokenOut:    tokenOut,
					HookAddress: metadata.GetHook(),
					PoolAddress: metadata.GetPool(),
					TickSpacing: metadata.GetTickSpacing(),
					Fee:         metadata.GetFee(),
					HookData:    metadata.GetHookData(),
				}
			}
		}
	}

	tokenInAddress, tokenOutAddress := NativeTokenAddress, NativeTokenAddress
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenInAfterWrap)] {
		tokenInAddress = common.HexToAddress(tokenInAfterWrap)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOutBeforeUnwrap)] {
		tokenOutAddress = common.HexToAddress(tokenOutBeforeUnwrap)
	}
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(tokenInAfterWrap == p.Info.Tokens[0], &priceLimit)

	return PoolMetaInfo{
		Router:            p.staticExtra.UniversalRouterAddress,
		Permit2Addr:       p.staticExtra.Permit2Address,
		TokenIn:           tokenInAddress,
		TokenOut:          tokenOutAddress,
		Fee:               p.staticExtra.Fee,
		TickSpacing:       p.staticExtra.TickSpacing,
		HookAddress:       p.staticExtra.HooksAddress,
		HookData:          []byte{},
		PriceLimit:        &priceLimit,
		TokenWrapMetadata: wrapMetadata,
	}
}
