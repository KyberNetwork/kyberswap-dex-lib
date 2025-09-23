package base

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	vault *vault.Vault
	swapper

	buffers      []*shared.ExtraBuffer
	bufferTokens []string
}

type swapper interface {
	BaseGas() int64
	OnSwap(param shared.PoolSwapParams) (*uint256.Int, error)
}

func NewPoolSimulator(entityPool entity.Pool, extra *shared.Extra, staticExtra *shared.StaticExtra, swapper swapper,
	hook hooks.IHook) (*PoolSimulator,
	error) {
	if err := validateExtra(extra); err != nil {
		return nil, err
	}

	if extra.Buffers == nil {
		extra.Buffers = make([]*shared.ExtraBuffer, len(entityPool.Tokens))
	}

	if hook == nil {
		switch staticExtra.HookType {
		case shared.DirectionalFeeHookType:
			hook = hooks.NewDirectionalFeeHook()
		case shared.FeeTakingHookType:
			hook = hooks.NewFeeTakingHook()
		case shared.VeBALFeeDiscountHookType:
			hook = hooks.NewVeBALFeeDiscountHook()
		default:
			hook = hooks.NewNoOpHook()
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens[:len(extra.BalancesLiveScaled18)], // remove placeholder buffer tokens
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves[:len(extra.BalancesLiveScaled18)],
				func(item string, index int) *big.Int { return bignumber.NewBig10(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},

		vault: vault.New(hook, extra.HooksConfig, extra.DecimalScalingFactors, extra.TokenRates,
			extra.BalancesLiveScaled18, extra.StaticSwapFeePercentage, extra.AggregateSwapFeePercentage),
		swapper: swapper,

		buffers:      extra.Buffers,
		bufferTokens: staticExtra.BufferTokens,
	}, nil
}

// ResolveToken resolves a token address to its index and whether it's an underlying token
// Returns: (index, isUnderlyingToken, error)
func (p *PoolSimulator) ResolveToken(token string) (int, bool, error) {
	// Try main tokens first
	if index := p.GetTokenIndex(token); index >= 0 {
		// Only return true if there's a valid buffer at this index
		// In this case the pool token is an underlying token with a buffer
		if index < len(p.buffers) && p.buffers[index] != nil {
			return index, true, nil
		}
		// If no valid buffer, return index with false (not underlying token)
		// In this case the pool token is either a wrapped token that can't be unwrapped or a vanilla ERC20
		return index, false, nil
	}

	// Try buffer tokens (these are the wrapped tokens if they exist)
	for i, bufferToken := range p.bufferTokens {
		if bufferToken == token {
			return i, false, nil
		}
	}

	return -1, false, shared.ErrInvalidToken
}

// isBufferSwap checks if this is a same-index underlying/wrapped token conversion
func (p *PoolSimulator) isBufferSwap(indexIn, indexOut int, isTokenInUnderlying, isTokenOutUnderlying bool) bool {
	return indexIn == indexOut && isTokenInUnderlying != isTokenOutUnderlying
}

// handleBufferConversion handles the conversion between underlying and wrapped tokens of the same index
func (p *PoolSimulator) handleBufferConversion(index int, amount *uint256.Int, isUnderlyingToken bool) (*uint256.Int,
	error) {
	if index >= len(p.buffers) || p.buffers[index] == nil {
		return nil, fmt.Errorf("buffer not found for token at index %d", index)
	}

	var convertedAmount *uint256.Int
	var err error

	if isUnderlyingToken {
		// Converting from underlying to wrapped: underlying -> shares -> wrapped
		convertedAmount, err = p.buffers[index].ConvertToShares(amount)
	} else {
		// Converting from wrapped to underlying: wrapped -> assets -> underlying
		convertedAmount, err = p.buffers[index].ConvertToAssets(amount)
	}

	if err != nil {
		return nil, err
	}

	return convertedAmount, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	tokenIn := tokenAmountIn.Token
	if tokenIn == tokenOut {
		return nil, shared.ErrInvalidToken
	}
	indexIn, isTokenInUnderlying, err := p.ResolveToken(tokenIn)
	if err != nil {
		return nil, err
	}
	indexOut, isTokenOutUnderlying, err := p.ResolveToken(tokenOut)
	if err != nil {
		return nil, err
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	// Check if this is a same-index underlying/wrapped token conversion
	if p.isBufferSwap(indexIn, indexOut, isTokenInUnderlying, isTokenOutUnderlying) {
		amountOut, err := p.handleBufferConversion(indexIn, amountIn, isTokenInUnderlying)
		if err != nil {
			return nil, err
		}

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: bignumber.ZeroBI, // No swap fee for direct conversions
			},
			SwapInfo: shared.SwapInfo{
				AggregateFee: bignumber.ZeroBI, // No aggregate fee for direct conversions
			},
			Gas: bufferGas,
		}, nil
	}

	gas := p.BaseGas()
	if isTokenInUnderlying {
		if indexIn >= len(p.buffers) || p.buffers[indexIn] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenIn, indexIn)
		}
		amountIn, err = p.buffers[indexIn].ConvertToShares(amountIn)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	amountOut, totalSwapFee, aggregateFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.ExactIn,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountIn,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	if isTokenOutUnderlying {
		if indexOut >= len(p.buffers) || p.buffers[indexOut] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenOut, indexOut)
		}
		amountOut, err = p.buffers[indexOut].ConvertToAssets(amountOut)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: totalSwapFee.ToBig(),
		},
		SwapInfo: shared.SwapInfo{
			AggregateFee: aggregateFee.ToBig(),
		},
		Gas: gas,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn
	tokenOut := tokenAmountOut.Token
	if tokenIn == tokenOut {
		return nil, shared.ErrInvalidToken
	}
	indexIn, isTokenInUnderlying, err := p.ResolveToken(tokenIn)
	if err != nil {
		return nil, err
	}
	indexOut, isTokenOutUnderlying, err := p.ResolveToken(tokenOut)
	if err != nil {
		return nil, err
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountOut
	}

	// Check if this is a same-index underlying/wrapped token conversion
	if p.isBufferSwap(indexIn, indexOut, isTokenInUnderlying, isTokenOutUnderlying) {
		amountIn, err := p.handleBufferConversion(indexOut, amountOut, isTokenOutUnderlying)
		if err != nil {
			return nil, err
		}

		return &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: amountIn.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: bignumber.ZeroBI, // No swap fee for direct conversions
			},
			SwapInfo: shared.SwapInfo{
				AggregateFee: bignumber.ZeroBI, // No aggregate fee for direct conversions
			},
			Gas: bufferGas,
		}, nil
	}

	gas := p.BaseGas()
	if isTokenOutUnderlying {
		if indexOut >= len(p.buffers) || p.buffers[indexOut] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenOut, indexOut)
		}
		amountOut, err = p.buffers[indexOut].ConvertToShares(amountOut)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	amountIn, totalSwapFee, aggregateSwapFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.ExactOut,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountOut,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	if isTokenInUnderlying {
		if indexIn >= len(p.buffers) || p.buffers[indexIn] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenIn, indexIn)
		}
		amountIn, err = p.buffers[indexIn].ConvertToAssets(amountIn)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: totalSwapFee.ToBig(),
		},
		SwapInfo: shared.SwapInfo{
			AggregateFee: aggregateSwapFee.ToBig(),
		},
		Gas: gas,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.vault = p.vault.CloneState()
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})

	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, isTokenInUnderlying, _ := p.ResolveToken(params.TokenAmountIn.Token)
	indexOut, isTokenOutUnderlying, _ := p.ResolveToken(params.TokenAmountOut.Token)
	// Buffer swaps do not affect pool reserves as they interact with ERC4626 and buffer tokens directly
	if p.isBufferSwap(indexIn, indexOut, isTokenInUnderlying, isTokenOutUnderlying) {
		return
	}

	swapInfo, ok := params.SwapInfo.(shared.SwapInfo)
	if !ok {
		return
	}

	amountIn := params.TokenAmountIn.Amount
	if isTokenInUnderlying {
		// If token in is underlying we must use the converted shares amount for the balance update
		convertedAmount, _ := p.buffers[indexIn].ConvertToShares(uint256.MustFromBig(params.TokenAmountIn.Amount))
		amountIn = convertedAmount.ToBig()
	}

	updatedRawBalanceIn := new(big.Int)
	updatedRawBalanceIn.Add(p.Info.Reserves[indexIn], amountIn)
	updatedRawBalanceIn.Sub(updatedRawBalanceIn, swapInfo.AggregateFee)
	p.Info.Reserves[indexIn] = updatedRawBalanceIn

	amountGivenRaw := uint256.MustFromBig(updatedRawBalanceIn)

	_, err := p.vault.UpdateLiveBalance(indexIn, amountGivenRaw, shared.RoundDown)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", p.GetExchange(), p.Info.Address)
		return
	}

	amountOut := params.TokenAmountOut.Amount
	if isTokenOutUnderlying {
		// If token out is underlying we must use the converted shares amount for the balance update
		convertedAmount, _ := p.buffers[indexOut].ConvertToShares(uint256.MustFromBig(params.TokenAmountOut.Amount))
		amountOut = convertedAmount.ToBig()
	}

	updatedRawBalanceOut := new(big.Int)
	updatedRawBalanceOut.Sub(p.Info.Reserves[indexOut], amountOut)
	p.Info.Reserves[indexOut] = updatedRawBalanceOut

	amountGivenRaw.SetFromBig(updatedRawBalanceOut)

	_, err = p.vault.UpdateLiveBalance(indexOut, amountGivenRaw, shared.RoundDown)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", p.GetExchange(), p.Info.Address)
		return
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	indexIn, isTokenInUnderlying, _ := p.ResolveToken(tokenIn)
	indexOut, isTokenOutUnderlying, _ := p.ResolveToken(tokenOut)
	if p.isBufferSwap(indexIn, indexOut, isTokenInUnderlying, isTokenOutUnderlying) {
		return shared.PoolMetaInfo{
			BufferSwap: p.bufferTokens[indexIn],
		}
	}
	return shared.PoolMetaInfo{
		BufferTokenIn:  p.bufferTokens[indexIn],
		BufferTokenOut: p.bufferTokens[indexOut],
	}
}

func (p *PoolSimulator) GetTokens() []string {
	return append(p.Info.Tokens, lo.Compact(p.bufferTokens)...)
}

func (p *PoolSimulator) CanSwap(address string) []string {
	// Check if address exists in pool tokens
	poolTokenIndex := p.GetTokenIndex(address)
	// Check if address exists in buffer tokens
	bufferTokenIndex := -1
	for i, bufferToken := range p.bufferTokens {
		if bufferToken == address {
			bufferTokenIndex = i
			break
		}
	}

	// Return nil if address doesn't exist in either collection
	if poolTokenIndex == -1 && bufferTokenIndex == -1 {
		return nil
	}

	// Collect all tokens (pool tokens + buffer tokens) excluding the input address
	var result []string

	// Add all pool tokens except the input address
	for _, token := range p.Info.Tokens {
		if token != address {
			result = append(result, token)
		}
	}

	// Add all buffer tokens except the input address
	for _, bufferToken := range p.bufferTokens {
		if bufferToken != address {
			result = append(result, bufferToken)
		}
	}

	return result
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	return lo.Filter(p.CanSwap(address), func(item string, _ int) bool {
		return lo.IndexOf(p.bufferTokens, item) < 0
	})
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if lo.IndexOf(p.bufferTokens, address) >= 0 {
		return []string{}
	}
	return p.CanSwap(address)
}
