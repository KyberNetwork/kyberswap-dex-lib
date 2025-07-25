package uniswapv4

import (
	"fmt"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
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
	}, nil
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	poolSim := p.PoolSimulator
	if p.hook == nil {
		return poolSim.CalcAmountOut(param)
	}

	beforeSwapHookParams := &BeforeSwapHookParams{
		ExactIn:         true,
		ZeroForOne:      p.Pool.GetTokenIndex(param.TokenAmountIn.Token) == 0,
		AmountSpecified: param.TokenAmountIn.Amount,
	}
	swapHookResult, err := p.hook.BeforeSwap(beforeSwapHookParams)
	if err != nil {
		return nil, err
	}

	// beforeSwap -> amountToSwap += hookDeltaSpecified;
	// for the case of calcAmountOut (exactIn) means amountToSwap(amountIn) supposed to be Negative, then turn to be TokenAmountIn.Sub
	// fot the case of calcAmountIn (exactOut) means amountToSwap(amountOut) supposed to be Positive, then turn to be TokenAmountOut.Add
	if swapHookResult != nil && swapHookResult.DeltaSpecific != nil {
		param.TokenAmountIn.Amount.Sub(param.TokenAmountIn.Amount, swapHookResult.DeltaSpecific)
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

	result, err := poolSim.CalcAmountOut(param)
	if err != nil {
		return nil, err
	}
	hookFee := p.hook.AfterSwap(&AfterSwapHookParams{
		BeforeSwapHookParams: beforeSwapHookParams,
		AmountIn:             param.TokenAmountIn.Amount,
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

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	poolSim := p.PoolSimulator
	if p.hook == nil {
		return poolSim.CalcAmountIn(param)
	}
	beforeSwapHookParams := &BeforeSwapHookParams{
		ExactIn:         false,
		ZeroForOne:      p.Pool.GetTokenIndex(param.TokenAmountOut.Token) == 1,
		AmountSpecified: param.TokenAmountOut.Amount,
	}

	swapHookResult, err := p.hook.BeforeSwap(beforeSwapHookParams)
	if err != nil {
		return nil, err
	}

	if swapHookResult != nil && swapHookResult.DeltaSpecific != nil {
		param.TokenAmountOut.Amount.Add(param.TokenAmountOut.Amount, swapHookResult.DeltaSpecific)
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

	result, err := poolSim.CalcAmountIn(param)
	if err != nil {
		return nil, err
	}

	hookFee := p.hook.AfterSwap(&AfterSwapHookParams{
		BeforeSwapHookParams: beforeSwapHookParams,
		AmountIn:             result.TokenAmountIn.Amount,
		AmountOut:            param.TokenAmountOut.Amount,
	})

	if swapHookResult != nil && swapHookResult.DeltaUnSpecific != nil {
		result.TokenAmountIn.Amount.Add(result.TokenAmountIn.Amount, swapHookResult.DeltaUnSpecific)
	}
	if hookFee != nil {
		result.TokenAmountIn.Amount.Add(result.TokenAmountIn.Amount, hookFee)
	}

	return result, err
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	tokenInAddress, tokenOutAddress := NativeTokenAddress, NativeTokenAddress
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0], &priceLimit)

	return PoolMetaInfo{
		Router:      p.staticExtra.UniversalRouterAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		TickSpacing: p.staticExtra.TickSpacing,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
		PriceLimit:  &priceLimit,
	}
}
