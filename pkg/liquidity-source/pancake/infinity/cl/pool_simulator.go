package cl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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
	if !ok && staticExtra.HasSwapPermissions {
		return nil, shared.ErrUnsupportedHook
	}

	// modify ticks before new pool simulator, some hooks will need this.
	// In the the original logic, we should do this logic in BeforeSwap in CalcAmountOut, but using here for simplicity.
	err := hook.ModifyTicks(context.Background(), extra.ExtraTickU256)
	if err != nil {
		return nil, err
	}

	var allowEmptyTicks bool
	v3PoolSimulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, chainID, extra.ExtraTickU256,
		allowEmptyTicks)
	if err != nil {
		return nil, err
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult,
	err error) {
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
				swapResult.TokenAmountOut.Amount.Sub(swapResult.TokenAmountOut.Amount,
					beforeSwapResult.DeltaUnspecified)
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
			err = ErrInvalidAmountOut
		}
	}()

	// If no hooks, just do swap
	poolSim := p.PoolSimulator
	if p.hook == nil {
		return p.PoolSimulator.CalcAmountOut(param)
	}

	tokenIn := param.TokenAmountIn.Token
	zeroForOne := p.GetTokenIndex(tokenIn) == 0
	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		ExactIn:         true,
		ZeroForOne:      zeroForOne,
		AmountSpecified: amountIn,
	}); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	if amountIn.Sub(amountIn, beforeSwapResult.DeltaSpecified).Sign() < 0 {
		return nil, ErrInvalidAmountIn
	}

	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		// clone
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if swapResult, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: param.TokenOut,
	}); err != nil {
		return nil, err
	}

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
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
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
			err = ErrInvalidAmountIn
		}
	}()

	poolSim := p.PoolSimulator
	if p.hook == nil {
		swapResult, err = poolSim.CalcAmountIn(param)
		return
	}

	tokenOut := param.TokenAmountOut.Token
	zeroForOne := p.GetTokenIndex(tokenOut) == 1
	amountOut := new(big.Int).Set(param.TokenAmountOut.Amount)

	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		ExactIn:         false,
		ZeroForOne:      zeroForOne,
		AmountSpecified: amountOut,
	}); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	if amountOut.Add(amountOut, beforeSwapResult.DeltaSpecified).Sign() < 0 {
		return nil, ErrInvalidAmountOut
	}

	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if swapResult, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		TokenIn: param.TokenIn,
	}); err != nil {
		return nil, err
	}

	if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
		BeforeSwapParams: &BeforeSwapParams{
			ExactIn:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: amountOut,
		},
		AmountIn:  swapResult.TokenAmountIn.Amount,
		AmountOut: amountOut,
	}); err != nil {
		return nil, fmt.Errorf("[AfterSwap] %w", err)
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
	}

	return
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
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

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAddress, tokenOutAddress := eth.AddressZero, eth.AddressZero
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}

	zeroForOne := strings.EqualFold(tokenIn, p.GetTokens()[0])
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(zeroForOne, &priceLimit)

	return PoolMetaInfo{
		Vault:       p.staticExtra.VaultAddress,
		PoolManager: p.staticExtra.PoolManagerAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		Parameters:  p.staticExtra.Parameters,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
		PriceLimit:  &priceLimit,
		SwapFee:     p.Info.SwapFee.Uint64(),
	}
}
