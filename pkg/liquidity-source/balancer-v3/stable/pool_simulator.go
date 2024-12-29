package stable

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

var (
	ErrInvalidSwapFeePercentage = errors.New("invalid swap fee percentage")
	ErrPoolIsPaused             = errors.New("pool is paused")
	ErrInvalidAmp               = errors.New("invalid amp")
	ErrNotTwoTokens             = errors.New("not two tokens")

	ErrTradeAmountTooSmall = errors.New("trade amount is too small")

	ErrVaultIsLocked = errors.New("vault is locked")
)

type PoolSimulator struct {
	poolpkg.Pool

	swapFeePercentage          *uint256.Int
	aggregateSwapFeePercentage *uint256.Int

	amplificationParameter *uint256.Int
	balancesLiveScaled18   []*uint256.Int
	decimalScalingFactors  []*uint256.Int
	tokenRates             []*uint256.Int

	isVaultLocked bool
	isPaused      bool

	vault string

	poolType    string
	poolVersion int
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
		Pool:                       poolpkg.Pool{Info: poolInfo},
		isPaused:                   extra.IsPaused,
		isVaultLocked:              extra.IsVaultLocked,
		swapFeePercentage:          extra.SwapFeePercentage,
		aggregateSwapFeePercentage: extra.AggregateSwapFeePercentage,
		amplificationParameter:     extra.AmplificationParameter,
		balancesLiveScaled18:       extra.BalancesLiveScaled18,
		tokenRates:                 extra.TokenRates,
		decimalScalingFactors:      extra.DecimalScalingFactors,
		vault:                      staticExtra.Vault,
		poolType:                   staticExtra.PoolType,
		poolVersion:                staticExtra.PoolVersion,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if p.isVaultLocked {
		return nil, ErrVaultIsLocked
	}

	if p.isPaused {
		return nil, ErrPoolIsPaused
	}

	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrTokenNotRegistered
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	amountIn, totalSwapFee, aggregateSwapFee, err := shared.Swap(shared.VaultSwapParams{
		IsExactIn:                  false,
		IndexIn:                    indexIn,
		IndexOut:                   indexOut,
		AmountGiven:                amountOut,
		DecimalScalingFactor:       p.decimalScalingFactors[indexOut],
		TokenRate:                  p.decimalScalingFactors[indexOut],
		AmplificationParameter:     p.amplificationParameter,
		SwapFeePercentage:          p.swapFeePercentage,
		AggregateSwapFeePercentage: p.aggregateSwapFeePercentage,
		BalancesLiveScaled18:       p.balancesLiveScaled18,
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

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vault,
		PoolType:      s.poolType,
		PoolVersion:   s.poolVersion,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
	}
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

	vaultParams := shared.VaultSwapParams{
		AmountGiven:          uint256.MustFromBig(updatedRawBalanceIn),
		DecimalScalingFactor: p.decimalScalingFactors[tokenIndexIn],
		TokenRate:            p.tokenRates[tokenIndexIn],
	}

	updatedLiveBalanceIn, err := shared.UpdateLiveBalance(vaultParams, shared.ROUND_DOWN)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for %v pool", DexType, p.Info.Address)
		return
	}
	p.balancesLiveScaled18[tokenIndexIn] = updatedLiveBalanceIn

	updatedRawBalanceOut := new(big.Int)
	updatedRawBalanceOut.Sub(p.Info.Reserves[tokenIndexOut], params.TokenAmountOut.Amount)
	p.Info.Reserves[tokenIndexOut] = updatedRawBalanceOut

	vaultParams.AmountGiven.SetFromBig(updatedRawBalanceOut)
	vaultParams.DecimalScalingFactor = p.decimalScalingFactors[tokenIndexOut]
	vaultParams.TokenRate = p.tokenRates[tokenIndexOut]

	updatedLiveBalanceOut, err := shared.UpdateLiveBalance(vaultParams, shared.ROUND_DOWN)
	if err != nil {
		logger.Warnf("[%s] failed to UpdateBalance for %v pool", DexType, p.Info.Address)
		return
	}
	p.balancesLiveScaled18[tokenIndexOut] = updatedLiveBalanceOut
}

func (p *PoolSimulator) computeBalance(tokenInIndex int, invariantRatio *uint256.Int) (*uint256.Int, error) {
	invariant, err := p.computeInvariant(shared.ROUND_UP)
	if err != nil {
		return nil, err
	}

	newInvariant, err := math.MulUp(invariant, invariantRatio)
	if err != nil {
		return nil, err
	}

	return math.StableMath.ComputeBalance(p.amplificationParameter, p.balancesLiveScaled18, newInvariant, tokenInIndex)
}

func (p *PoolSimulator) OnSwap(isExactIn bool, indexIn, indexOut int, amountInScaled18 *uint256.Int) (*uint256.Int, error) {
	invariant, err := p.computeInvariant(shared.ROUND_DOWN)
	if err != nil {
		return nil, err
	}

	var amountOutScaled18 *uint256.Int
	if isExactIn {
		amountOutScaled18, err = math.StableMath.ComputeOutGivenExactIn(
			p.amplificationParameter,
			p.balancesLiveScaled18,
			indexIn,
			indexOut,
			amountInScaled18,
			invariant,
		)
	} else {
		amountOutScaled18, err = math.StableMath.ComputeInGivenExactOut(
			p.amplificationParameter,
			p.balancesLiveScaled18,
			indexIn,
			indexOut,
			amountInScaled18,
			invariant,
		)
	}
	if err != nil {
		return nil, err
	}

	return amountOutScaled18, nil
}

func (p *PoolSimulator) computeInvariant(rounding shared.Rounding) (*uint256.Int, error) {
	invariant, err := math.StableMath.ComputeInvariant(p.amplificationParameter, p.balancesLiveScaled18)
	if err != nil {
		return nil, err
	}

	if invariant.Sign() > 0 && rounding == shared.ROUND_UP {
		return invariant.AddUint64(invariant, 1), nil
	}

	return invariant, nil
}

func (p *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if p.isVaultLocked {
		return nil, ErrVaultIsLocked
	}

	if p.isPaused {
		return nil, ErrPoolIsPaused
	}

	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrTokenNotRegistered
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	amountOut, totalSwapFee, aggregateFee, err := shared.Swap(shared.VaultSwapParams{
		IsExactIn:                  true,
		IndexIn:                    indexIn,
		IndexOut:                   indexOut,
		AmountGiven:                amountIn,
		DecimalScalingFactor:       p.decimalScalingFactors[indexIn],
		TokenRate:                  p.decimalScalingFactors[indexIn],
		AmplificationParameter:     p.amplificationParameter,
		SwapFeePercentage:          p.swapFeePercentage,
		AggregateSwapFeePercentage: p.aggregateSwapFeePercentage,
		BalancesLiveScaled18:       p.balancesLiveScaled18,
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

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.swapFeePercentage = p.swapFeePercentage.Clone()
	cloned.aggregateSwapFeePercentage = p.aggregateSwapFeePercentage.Clone()
	cloned.amplificationParameter = p.amplificationParameter.Clone()
	cloned.balancesLiveScaled18 = lo.Map(p.balancesLiveScaled18, func(v *uint256.Int, _ int) *uint256.Int {
		return new(uint256.Int).Set(v)
	})
	cloned.decimalScalingFactors = lo.Map(p.decimalScalingFactors, func(v *uint256.Int, _ int) *uint256.Int {
		return new(uint256.Int).Set(v)
	})
	cloned.tokenRates = lo.Map(p.tokenRates, func(v *uint256.Int, _ int) *uint256.Int {
		return new(uint256.Int).Set(v)
	})
	return &cloned
}
