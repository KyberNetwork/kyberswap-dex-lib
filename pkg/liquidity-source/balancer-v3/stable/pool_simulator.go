package stable

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

var (
	ErrInvalidSwapFeePercentage = errors.New("invalid swap fee percentage")
	ErrInvalidAmp               = errors.New("invalid amp")
	ErrNotTwoTokens             = errors.New("not two tokens")
)

type PoolSimulator struct {
	poolpkg.Pool

	vault      *vault.Vault
	currentAmp *uint256.Int

	hooksConfig shared.HooksConfig

	isVaultPaused        bool
	isPoolPaused         bool
	isPoolInRecoveryMode bool

	vaultAddress string
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		tokens[idx] = entityPool.Tokens[idx].Address
		reserves[idx] = bignumber.NewBig10(entityPool.Reserves[idx])
	}

	// Need to detect the current hook type of pool
	hook := hooks.NewBaseHook()

	vault := vault.New(hook, extra.HooksConfig, extra.IsPoolInRecoveryMode, extra.DecimalScalingFactors, extra.TokenRates,
		extra.BalancesLiveScaled18, extra.StaticSwapFeePercentage, extra.AggregateSwapFeePercentage)

	poolInfo := poolpkg.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:                 poolpkg.Pool{Info: poolInfo},
		isVaultPaused:        extra.IsVaultPaused,
		isPoolPaused:         extra.IsPoolPaused,
		isPoolInRecoveryMode: extra.IsPoolInRecoveryMode,
		vault:                vault,
		hooksConfig:          extra.HooksConfig,
		currentAmp:           extra.AmplificationParameter,
		vaultAddress:         staticExtra.Vault,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if p.isVaultPaused {
		return nil, shared.ErrVaultIsPaused
	}

	if p.isPoolPaused {
		return nil, shared.ErrPoolIsPaused
	}

	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrTokenNotRegistered
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	amountOut, totalSwapFee, aggregateFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.EXACT_IN,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountIn,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: totalSwapFee.ToBig(),
		},
		SwapInfo: SwapInfo{
			AggregateFee: aggregateFee.ToBig(),
		},
		Gas: defaultGas.Swap,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if p.isVaultPaused {
		return nil, shared.ErrVaultIsPaused
	}

	if p.isPoolPaused {
		return nil, shared.ErrPoolIsPaused
	}

	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrTokenNotRegistered
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountOut
	}

	amountIn, totalSwapFee, aggregateSwapFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.EXACT_OUT,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountOut,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: totalSwapFee.ToBig(),
		},
		SwapInfo: SwapInfo{
			AggregateFee: aggregateSwapFee.ToBig(),
		},
		Gas: defaultGas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	tokenIndexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	tokenIndexOut := p.GetTokenIndex(params.TokenAmountOut.Token)

	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	updatedRawBalanceIn := new(big.Int)
	updatedRawBalanceIn.Add(p.Info.Reserves[tokenIndexIn], params.TokenAmountIn.Amount)
	updatedRawBalanceIn.Sub(updatedRawBalanceIn, swapInfo.AggregateFee)
	p.Info.Reserves[tokenIndexIn] = updatedRawBalanceIn

	amountGivenRaw := uint256.MustFromBig(updatedRawBalanceIn)

	_, err := p.vault.UpdateLiveBalance(tokenIndexIn, amountGivenRaw, shared.ROUND_DOWN)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for %v pool", DexType, p.Info.Address)
		return
	}

	updatedRawBalanceOut := new(big.Int)
	updatedRawBalanceOut.Sub(p.Info.Reserves[tokenIndexOut], params.TokenAmountOut.Amount)
	p.Info.Reserves[tokenIndexOut] = updatedRawBalanceOut

	amountGivenRaw.SetFromBig(updatedRawBalanceOut)

	_, err = p.vault.UpdateLiveBalance(tokenIndexOut, amountGivenRaw, shared.ROUND_DOWN)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for %v pool", DexType, p.Info.Address)
		return
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vaultAddress,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
	}
}

func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (*uint256.Int, error) {
	invariant, err := p.computeInvariant(param.BalancesLiveScaled18, shared.ROUND_DOWN)
	if err != nil {
		return nil, err
	}

	var amountOutScaled18 *uint256.Int
	if param.Kind == shared.EXACT_IN {
		amountOutScaled18, err = math.StableMath.ComputeOutGivenExactIn(
			p.currentAmp,
			param.BalancesLiveScaled18,
			param.IndexIn,
			param.IndexOut,
			param.AmountGivenScaled18,
			invariant,
		)
	} else {
		amountOutScaled18, err = math.StableMath.ComputeInGivenExactOut(
			p.currentAmp,
			param.BalancesLiveScaled18,
			param.IndexIn,
			param.IndexOut,
			param.AmountGivenScaled18,
			invariant,
		)
	}
	if err != nil {
		return nil, err
	}

	return amountOutScaled18, nil
}

func (p *PoolSimulator) computeInvariant(balancesLiveScaled18 []*uint256.Int, rounding shared.Rounding) (*uint256.Int, error) {
	invariant, err := math.StableMath.ComputeInvariant(p.currentAmp, balancesLiveScaled18)
	if err != nil {
		return nil, err
	}

	if invariant.Sign() > 0 && rounding == shared.ROUND_UP {
		return invariant.AddUint64(invariant, 1), nil
	}

	return invariant, nil
}

func (p *PoolSimulator) CloneState() poolpkg.IPoolSimulator {
	cloned := *p
	cloned.vault = p.vault.CloneState()

	return &cloned
}
