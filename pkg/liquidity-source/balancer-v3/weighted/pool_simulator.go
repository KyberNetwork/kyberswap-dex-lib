package weighted

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

type PoolSimulator struct {
	poolpkg.Pool

	vault             *vault.Vault
	normalizedWeights []*uint256.Int

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
	if staticExtra.DefaultHook != "" && !hooks.IsHookSupported(staticExtra.DefaultHook) {
		logger.Warnf(
			"[%s] Pool Address: %s | Warning: defaultHook is not supported => falling back to BaseHook",
			DexType,
			entityPool.Address,
		)
	}

	var hook hooks.IHook
	switch staticExtra.DefaultHook {
	case hooks.DirectionalFeeHookType:
		hook = hooks.NewDirectionalFeeHook(extra.StaticSwapFeePercentage)
	default:
		hook = hooks.NewBaseHook()
	}

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
		normalizedWeights:    extra.NormalizedWeights,
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
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", DexType, p.Info.Address)
		return
	}

	updatedRawBalanceOut := new(big.Int)
	updatedRawBalanceOut.Sub(p.Info.Reserves[tokenIndexOut], params.TokenAmountOut.Amount)
	p.Info.Reserves[tokenIndexOut] = updatedRawBalanceOut

	amountGivenRaw.SetFromBig(updatedRawBalanceOut)

	_, err = p.vault.UpdateLiveBalance(tokenIndexOut, amountGivenRaw, shared.ROUND_DOWN)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for pool %s", DexType, p.Info.Address)
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

// https://etherscan.io/address/0xb9b144b5678ff6527136b2c12a86c9ee5dd12a85#code#F1#L150
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (amountOutScaled18 *uint256.Int, err error) {
	balanceTokenInScaled18 := param.BalancesLiveScaled18[param.IndexIn]
	balanceTokenOutScaled18 := param.BalancesLiveScaled18[param.IndexOut]

	weightIn, err := p.getNormalizedWeight(param.IndexIn)
	if err != nil {
		return nil, err
	}

	weightOut, err := p.getNormalizedWeight(param.IndexOut)
	if err != nil {
		return nil, err
	}

	if param.Kind == shared.EXACT_IN {
		amountOutScaled18, err = math.WeightedMath.ComputeOutGivenExactIn(
			balanceTokenInScaled18,
			weightIn,
			balanceTokenOutScaled18,
			weightOut,
			param.AmountGivenScaled18,
		)
	} else {
		amountOutScaled18, err = math.WeightedMath.ComputeInGivenExactOut(
			balanceTokenInScaled18,
			weightIn,
			balanceTokenOutScaled18,
			weightOut,
			param.AmountGivenScaled18,
		)
	}
	if err != nil {
		return nil, err
	}

	return
}

func (p *PoolSimulator) getNormalizedWeight(tokenIndex int) (*uint256.Int, error) {
	if tokenIndex > len(p.normalizedWeights) {
		return nil, ErrInvalidToken
	}

	return p.normalizedWeights[tokenIndex], nil
}

func (p *PoolSimulator) CloneState() poolpkg.IPoolSimulator {
	cloned := *p
	cloned.vault = p.vault.CloneState()
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})

	return &cloned
}
