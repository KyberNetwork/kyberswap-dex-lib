package gyroeclp

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPoolPaused         = errors.New("pool is paused")
	ErrTokenInIsNotToken0 = errors.New("TOKEN_IN_IS_NOT_TOKEN_0")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
)

type PoolSimulator struct {
	pool.Pool

	paused bool

	_paramsAlpha  *int256.Int
	_paramsBeta   *int256.Int
	_paramsC      *int256.Int
	_paramsS      *int256.Int
	_paramsLambda *int256.Int
	_tauAlphaX    *int256.Int
	_tauAlphaY    *int256.Int
	_tauBetaX     *int256.Int
	_tauBetaY     *int256.Int
	_u            *int256.Int
	_v            *int256.Int
	_w            *int256.Int
	_z            *int256.Int
	_dSq          *int256.Int

	swapFeePercentage *uint256.Int
	scalingFactors    []*uint256.Int

	vault  string
	poolID string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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

	scalingFactors := getScalingFactors(staticExtra.PoolTypeVer, staticExtra.TokenDecimals, extra.TokenRates)

	poolInfo := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:              pool.Pool{Info: poolInfo},
		paused:            extra.Paused,
		_paramsAlpha:      extra.ParamsAlpha,
		_paramsBeta:       extra.ParamsBeta,
		_paramsC:          extra.ParamsC,
		_paramsS:          extra.ParamsS,
		_paramsLambda:     extra.ParamsLambda,
		_tauAlphaX:        extra.TauAlphaX,
		_tauAlphaY:        extra.TauAlphaY,
		_tauBetaX:         extra.TauBetaX,
		_tauBetaY:         extra.TauBetaY,
		_u:                extra.U,
		_v:                extra.V,
		_w:                extra.W,
		_z:                extra.Z,
		_dSq:              extra.DSq,
		swapFeePercentage: extra.SwapFeePercentage,
		scalingFactors:    scalingFactors,
		vault:             staticExtra.Vault,
		poolID:            staticExtra.PoolID,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	var tokenInIsToken0 bool
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn == 0 && indexOut == 1 {
		tokenInIsToken0 = true
	} else if indexIn == 1 && indexOut == 0 {
		tokenInIsToken0 = false
	} else {
		return nil, ErrTokenInIsNotToken0
	}

	scalingFactorTokenIn := s._scalingFactor(tokenInIsToken0)
	scalingFactorTokenOut := s._scalingFactor(!tokenInIsToken0)

	balanceTokenIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}
	balanceTokenOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenIn, err := _upscale(balanceTokenIn, scalingFactorTokenIn)
	if err != nil {
		return nil, ErrInvalidReserve
	}
	balanceTokenOut, err = _upscale(balanceTokenOut, scalingFactorTokenOut)
	if err != nil {
		return nil, ErrInvalidReserve
	}

	balances := s._balancesFromTokenInOut(balanceTokenIn, balanceTokenOut, tokenInIsToken0)

	eclpParams, derivedECLPParams := s.reconstructECLPParams()

	invariant := &vector2{}
	{
		currentInvariant, invErr, err := GyroECLPMath.calculateInvariantWithError(
			balances, eclpParams, derivedECLPParams,
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

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}
	feeAmount, err := math.GyroFixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}
	amountInAfterFee, err := math.GyroFixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}
	amountInAfterFee, err = _upscale(amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	amountOut, err := GyroECLPMath.calcOutGivenIn(
		balances, amountInAfterFee, tokenInIsToken0, eclpParams, derivedECLPParams, invariant,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err = _downscaleDown(amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            defaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) reconstructECLPParams() (*params, *derivedParams) {
	p := &params{
		Alpha:  s._paramsAlpha,
		Beta:   s._paramsBeta,
		C:      s._paramsC,
		S:      s._paramsS,
		Lambda: s._paramsLambda,
	}

	dp := &derivedParams{
		TauAlpha: &vector2{
			X: s._tauAlphaX,
			Y: s._tauAlphaY,
		},
		TauBeta: &vector2{
			X: s._tauBetaX,
			Y: s._tauBetaY,
		},
		U:   s._u,
		V:   s._v,
		W:   s._w,
		Z:   s._z,
		DSq: s._dSq,
	}

	return p, dp

}

func (s *PoolSimulator) _balancesFromTokenInOut(
	balanceTokenIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	tokenInIsToken0 bool,
) []*uint256.Int {
	balances := make([]*uint256.Int, 2)
	if tokenInIsToken0 {
		balances[0] = balanceTokenIn
		balances[1] = balanceTokenOut
	} else {
		balances[0] = balanceTokenOut
		balances[1] = balanceTokenIn
	}

	return balances
}

func (s *PoolSimulator) _scalingFactor(token0 bool) *uint256.Int {
	if token0 {
		return s.scalingFactors[0]
	}
	return s.scalingFactors[1]
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	for idx, token := range s.Info.Tokens {
		if token == params.TokenAmountIn.Token {
			s.Info.Reserves[idx] = new(big.Int).Add(
				s.Info.Reserves[idx],
				params.TokenAmountIn.Amount,
			)
		}

		if token == params.TokenAmountOut.Token {
			s.Info.Reserves[idx] = new(big.Int).Sub(
				s.Info.Reserves[idx],
				params.TokenAmountOut.Amount,
			)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vault,
		PoolID:        s.poolID,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
	}
}

func _upscale(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.MulDown(amount, scalingFactor)
}

func _downscaleDown(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.DivDown(amount, scalingFactor)
}

func getScalingFactors(poolTypeVer int, tokenDecimals []int, tokenRates []*uint256.Int) []*uint256.Int {
	// NOTE: token rates are achieved by calling `IRateProvider` contracts, some of them
	// calculate the rate using block.timestamp, so the rate can be changed over time and
	// out of sync between actual onchain execution and previous simulation.

	if poolTypeVer == poolTypeVer1 {
		return []*uint256.Int{
			computeScalingFactor(tokenDecimals[0]),
			computeScalingFactor(tokenDecimals[1]),
		}
	}

	f0, _ := math.GyroFixedPoint.MulDown(computeScalingFactor(tokenDecimals[0]), tokenRates[0])
	f1, _ := math.GyroFixedPoint.MulDown(computeScalingFactor(tokenDecimals[1]), tokenRates[1])
	return []*uint256.Int{f0, f1}
}

func computeScalingFactor(decimal int) *uint256.Int {
	return new(uint256.Int).Mul(
		number.TenPow(uint8(18-decimal)),
		math.GyroFixedPoint.ONE,
	)
}
