package uniswapv4

import (
	"fmt"
	"maps"
	"math/big"
	"slices"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

	var allowEmptyTicks bool
	switch hook.GetExchange() {
	case valueobject.ExchangeUniswapV4BunniV2, valueobject.ExchangeUniswapV4Deli:
		allowEmptyTicks = true
	}

	v3PoolSimulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, chainID, extra.ExtraTickU256, allowEmptyTicks)
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult, err error) {
	originalTokenIn, originalTokenOut := param.TokenAmountIn.Token, param.TokenOut
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult

	defer func() { // modify result before return
		if swapResult == nil {
			return
		}
		v4SwapInfo := SwapInfo{
			PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo),
		}

		if swapResult.TokenAmountOut != nil {
			swapResult.TokenAmountOut.Token = originalTokenOut

			if beforeSwapResult != nil {
				swapResult.TokenAmountOut.Amount.Sub(swapResult.TokenAmountOut.Amount, beforeSwapResult.DeltaUnspecified)
				swapResult.Gas += beforeSwapResult.Gas
				v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
			}

			if afterSwapResult != nil {
				swapResult.TokenAmountOut.Amount.Sub(swapResult.TokenAmountOut.Amount, afterSwapResult.HookFee)
				swapResult.Gas += afterSwapResult.Gas
			}
		}
		swapResult.SwapInfo = v4SwapInfo

		if swapResult.RemainingTokenAmountIn != nil {
			swapResult.RemainingTokenAmountIn.Token = originalTokenIn
		}

		swapResult.Gas += wrapAdditionalGas

		if swapResult.TokenAmountOut.Amount.Sign() < 0 {
			swapResult = nil
			err = errors.New("amount out is invalid")
		}
	}()

	// Wrap/unwrap tokens if needed and calculate wrap gas
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

	// If no hooks, just do swap
	poolSim := p.PoolSimulator
	if p.hook == nil {
		return p.PoolSimulator.CalcAmountOut(param)
	}

	tokenIn := param.TokenAmountIn.Token
	zeroForOne := p.GetTokenIndex(tokenIn) == 0
	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			ExactIn:         true,
			ZeroForOne:      zeroForOne,
			AmountSpecified: amountIn,
		})
		if err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		}
		if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		amountIn.Sub(amountIn, beforeSwapResult.DeltaSpecified)
		if amountIn.Sign() < 0 {
			return nil, errors.New("[BeforeSwap] amount in is negative")
		}

		if beforeSwapResult.SwapFee >= FeeMax {
			return nil, errors.New("[BeforeSwap] swap fee is greater than max fee")
		} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
			cloned := *poolSim
			clonedV3Pool := *poolSim.V3Pool
			cloned.V3Pool = &clonedV3Pool
			cloned.V3Pool.Fee = beforeSwapResult.SwapFee
			poolSim = &cloned
		}
	}

	swapResult, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: param.TokenOut,
	})
	if err != nil {
		return nil, err
	}

	if p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				ExactIn:         true,
				ZeroForOne:      zeroForOne,
				AmountSpecified: amountIn,
			},
			AmountIn:  amountIn,
			AmountOut: swapResult.TokenAmountOut.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		}
		if err := ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	return
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (swapResult *pool.CalcAmountInResult, err error) {
	originalTokenOut, originalTokenIn := param.TokenAmountOut.Token, param.TokenIn
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult

	defer func() { // modify result before return
		if swapResult == nil {
			return
		}
		v4SwapInfo := SwapInfo{
			PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo),
		}

		if swapResult.TokenAmountIn != nil {
			swapResult.TokenAmountIn.Token = originalTokenIn

			if beforeSwapResult != nil {
				swapResult.TokenAmountIn.Amount.Add(swapResult.TokenAmountIn.Amount, beforeSwapResult.DeltaUnspecified)
				swapResult.Gas += beforeSwapResult.Gas
				v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
			}

			if afterSwapResult != nil {
				swapResult.TokenAmountIn.Amount.Add(swapResult.TokenAmountIn.Amount, afterSwapResult.HookFee)
				swapResult.Gas += afterSwapResult.Gas
			}
		}
		swapResult.SwapInfo = v4SwapInfo

		if swapResult.RemainingTokenAmountOut != nil {
			swapResult.RemainingTokenAmountOut.Token = originalTokenOut
		}

		swapResult.Gas += wrapAdditionalGas

		if swapResult.TokenAmountIn.Amount.Sign() < 0 {
			swapResult = nil
			err = errors.New("amount in is invalid")
		}
	}()

	// Wrap/unwrap tokens if needed and calculate wrap gas
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
		swapResult, err = poolSim.CalcAmountIn(param)
		return
	}

	tokenOut := param.TokenAmountOut.Token
	zeroForOne := p.GetTokenIndex(tokenOut) == 1
	amountOut := new(big.Int).Set(param.TokenAmountOut.Amount)

	if p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			ExactIn:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: amountOut,
		})
		if err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		}

		if err := ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		amountOut.Add(amountOut, beforeSwapResult.DeltaSpecified)
		if amountOut.Sign() < 0 {
			return nil, errors.New("[BeforeSwap] amount out is negative")
		}

		if beforeSwapResult.SwapFee >= FeeMax {
			return nil, errors.New("[BeforeSwap] swap fee is greater than max fee")
		} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
			cloned := *poolSim
			clonedV3Pool := *poolSim.V3Pool
			cloned.V3Pool = &clonedV3Pool
			cloned.V3Pool.Fee = beforeSwapResult.SwapFee
			poolSim = &cloned
		}
	}

	swapResult, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		TokenIn: param.TokenIn,
	})
	if err != nil {
		return nil, err
	}

	if p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				ExactIn:         false,
				ZeroForOne:      zeroForOne,
				AmountSpecified: amountOut,
			},
			AmountIn:  swapResult.TokenAmountIn.Amount,
			AmountOut: amountOut,
		})
		if err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		}

		if err := ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	return
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
	seen := make(map[string]struct{})
	tokens := make([]string, 0, len(p.Info.Tokens))

	for _, token := range p.Info.Tokens {
		if _, exists := seen[token]; !exists {
			seen[token] = struct{}{}
			tokens = append(tokens, token)
		}

		for _, wrapper := range p.tokenWrappers {
			if metadata, isWrapped := wrapper.IsWrapped(p.chainID, token); isWrapped {
				unwrappedToken := metadata.GetUnwrapToken()
				if _, exists := seen[unwrappedToken]; !exists {
					seen[unwrappedToken] = struct{}{}
					tokens = append(tokens, unwrappedToken)
				}
			}
		}
	}

	return tokens
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	if cloned.hook != nil {
		cloned.hook = p.hook.CloneState()
		if _, ok := cloned.hook.(*BaseHook); ok {
			if _, ok = p.hook.(*BaseHook); !ok {
				cloned.hook = p.hook
			}
		}
	}
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.SwapInfo == nil {
		return
	}
	v4SwapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	if p.hook != nil {
		p.hook.UpdateBalance(v4SwapInfo.HookSwapInfo)
	}
	params.SwapInfo = v4SwapInfo.PoolSwapInfo
	p.PoolSimulator.UpdateBalance(params)
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAfterWrap, tokenOutBeforeUnwrap := tokenIn, tokenOut
	var wrapMetadata *TokenWrapMetadata

	if p.GetTokenIndex(tokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			if metadata, canWrap := wrapper.CanWrap(p.chainID, tokenIn); canWrap {
				tokenInAfterWrap = metadata.GetWrapToken()
				if metadata.IsUnwrapNative() {
					tokenIn = NativeTokenAddress.Hex()
				}

				wrapMetadata = &TokenWrapMetadata{WrapInfo: &WrapInfo{
					TokenIn:     tokenIn,
					TokenOut:    metadata.GetWrapToken(),
					HookAddress: metadata.GetHook(),
					PoolAddress: metadata.GetPool(),
					TickSpacing: metadata.GetTickSpacing(),
					Fee:         metadata.GetFee(),
					HookData:    metadata.GetHookData(),
				}}
				break
			}
		}
	}

	if p.GetTokenIndex(tokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			if metadata, canUnwrap := wrapper.CanWrap(p.chainID, tokenOut); canUnwrap {
				tokenOutBeforeUnwrap = metadata.GetWrapToken()
				if metadata.IsUnwrapNative() {
					tokenOut = NativeTokenAddress.Hex()
				}

				if wrapMetadata == nil {
					wrapMetadata = &TokenWrapMetadata{}
				}
				wrapMetadata.UnwrapInfo = &WrapInfo{
					TokenIn:     tokenOutBeforeUnwrap,
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
		HookData:          p.hook.GetHookData(),
		PriceLimit:        &priceLimit,
		TokenWrapMetadata: wrapMetadata,
	}
}
