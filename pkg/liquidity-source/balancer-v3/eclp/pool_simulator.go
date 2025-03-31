package eclp

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type PoolSimulator struct {
	pool.Pool

	vault *vault.Vault

	hooksConfig shared.HooksConfig

	buffers              []*shared.ExtraBuffer
	isVaultPaused        bool
	isPoolPaused         bool
	isPoolInRecoveryMode bool
	bufferTokens         []string
	eclpParams           ECLPParams
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.Buffers == nil {
		extra.Buffers = make([]*shared.ExtraBuffer, len(entityPool.Tokens))
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var hook hooks.IHook
	switch staticExtra.HookType {
	case shared.DirectionalFeeHookType:
		hook = hooks.NewDirectionalFeeHook(extra.StaticSwapFeePercentage)
	case shared.FeeTakingHookType:
		hook = hooks.NewFeeTakingHook()
	case shared.StableSurgeHookType:
		hook = hooks.NewStableSurgeHook()
	case shared.VeBALFeeDiscountHookType:
		hook = hooks.NewVeBALFeeDiscountHook()
	default:
		hook = hooks.NewNoOpHook()
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
		buffers:              extra.Buffers,
		isVaultPaused:        extra.IsVaultPaused,
		isPoolPaused:         extra.IsPoolPaused,
		isPoolInRecoveryMode: extra.IsPoolInRecoveryMode,
		vault: vault.New(hook, extra.HooksConfig, extra.DecimalScalingFactors,
			extra.TokenRates, extra.BalancesLiveScaled18, extra.StaticSwapFeePercentage,
			extra.AggregateSwapFeePercentage),
		hooksConfig: extra.HooksConfig,
		eclpParams:  extra.ECLPParams,
		// vaultAddress: staticExtra.Vault,
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

	gas := baseGas
	if bufferIn := p.buffers[indexIn]; bufferIn != nil {
		amountIn = bufferIn.ConvertToShares(amountIn)
		gas += bufferGas
	}
	ai := amountIn.String()
	fmt.Print(ai)
	amountOut, totalSwapFee, aggregateFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.EXACT_IN,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountIn,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	if bufferOut := p.buffers[indexOut]; bufferOut != nil {
		amountOut = bufferOut.ConvertToAssets(amountOut)
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

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	tokenInIdx, tokenOutIdx := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	return shared.PoolMetaInfo{
		BufferTokenIn:  p.bufferTokens[tokenInIdx],
		BufferTokenOut: p.bufferTokens[tokenOutIdx],
	}
}

// OnSwap https://arbiscan.io/address/0xc09a98b0138d8cfceff0e4ef672e8bd30ec6eda9#code#F1#L156
func (p *PoolSimulator) OnSwap(param shared.PoolSwapParams) (amountOutScaled18 *uint256.Int, err error) {
	eclpParams, derivedECLPParams := p.reconstructECLPParams()
	invariant := &vector2{}
	{
		currentInvariant, invErr, err := GyroECLPMath.calculateInvariantWithError(
			param.BalancesLiveScaled18, eclpParams, derivedECLPParams,
		)
		if err != nil {
			return nil, err
		}

		invariant.X = new(int256.Int).Add(
			currentInvariant,
			new(int256.Int).Mul(GyroECLPMath._number_2, invErr),
		)

		invariant.Y = currentInvariant
	}

	if param.Kind == shared.EXACT_IN {
		amountOutScaled18, err = GyroECLPMath.calcOutGivenIn(
			param.BalancesLiveScaled18,
			param.AmountGivenScaled18,
			param.IndexIn == 0,
			eclpParams,
			derivedECLPParams,
			invariant,
		)
	} else {
		// TODO: implement calcInGivenOut
		// amountOutScaled18, err = GyroECLPMath.calcInGivenOut(
		// 	param.BalancesLiveScaled18,
		// 	param.AmountGivenScaled18,
		// 	param.IndexIn == 0,
		// 	eclpParams,
		// 	derivedECLPParams,
		// 	invariant,
		// )
		return nil, errors.New("not implemented")
	}

	if err != nil {
		return nil, err
	}

	return
}

func (p *PoolSimulator) reconstructECLPParams() (*params, *derivedParams) {
	params := &params{
		Alpha:  p.eclpParams.Params.Alpha,
		Beta:   p.eclpParams.Params.Beta,
		C:      p.eclpParams.Params.C,
		S:      p.eclpParams.Params.S,
		Lambda: p.eclpParams.Params.Lambda,
	}

	dp := &derivedParams{
		TauAlpha: &vector2{
			X: p.eclpParams.D.TauAlpha.X,
			Y: p.eclpParams.D.TauAlpha.Y,
		},
		TauBeta: &vector2{
			X: p.eclpParams.D.TauBeta.X,
			Y: p.eclpParams.D.TauBeta.Y,
		},
		U:   p.eclpParams.D.U,
		V:   p.eclpParams.D.V,
		W:   p.eclpParams.D.W,
		Z:   p.eclpParams.D.Z,
		DSq: p.eclpParams.D.DSq,
	}

	return params, dp
}
