package base

import (
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
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
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

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	gas := p.BaseGas()
	var err error
	if bufferIn := p.buffers[indexIn]; bufferIn != nil {
		amountIn, err = bufferIn.ConvertToShares(amountIn)
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

	if bufferOut := p.buffers[indexOut]; bufferOut != nil {
		amountOut, err = bufferOut.ConvertToAssets(amountOut)
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
			Token:  tokenAmountIn.Token,
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

	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountOut
	}

	gas := p.BaseGas()
	var err error
	if bufferOut := p.buffers[indexOut]; bufferOut != nil {
		amountOut, err = bufferOut.ConvertToShares(amountOut)
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

	if bufferIn := p.buffers[indexIn]; bufferIn != nil {
		amountIn, err = bufferIn.ConvertToAssets(amountIn)
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
	tokenIndexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	tokenIndexOut := p.GetTokenIndex(params.TokenAmountOut.Token)

	swapInfo, ok := params.SwapInfo.(shared.SwapInfo)
	if !ok {
		return
	}

	updatedRawBalanceIn := new(big.Int)
	updatedRawBalanceIn.Add(p.Info.Reserves[tokenIndexIn], params.TokenAmountIn.Amount)
	updatedRawBalanceIn.Sub(updatedRawBalanceIn, swapInfo.AggregateFee)
	p.Info.Reserves[tokenIndexIn] = updatedRawBalanceIn

	amountGivenRaw := uint256.MustFromBig(updatedRawBalanceIn)

	_, err := p.vault.UpdateLiveBalance(tokenIndexIn, amountGivenRaw, shared.RoundDown)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", p.GetExchange(), p.Info.Address)
		return
	}

	updatedRawBalanceOut := new(big.Int)
	updatedRawBalanceOut.Sub(p.Info.Reserves[tokenIndexOut], params.TokenAmountOut.Amount)
	p.Info.Reserves[tokenIndexOut] = updatedRawBalanceOut

	amountGivenRaw.SetFromBig(updatedRawBalanceOut)

	_, err = p.vault.UpdateLiveBalance(tokenIndexOut, amountGivenRaw, shared.RoundDown)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", p.GetExchange(), p.Info.Address)
		return
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	tokenInIdx, tokenOutIdx := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	return shared.PoolMetaInfo{
		BufferTokenIn:  p.bufferTokens[tokenInIdx],
		BufferTokenOut: p.bufferTokens[tokenOutIdx],
	}
}
