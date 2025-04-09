package weighted

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidNormalizedWeights   = errors.New("invalid normalizedWeights")
	ErrBasePoolIsNil              = errors.New("base pool is nil")
	ErrSameBasePoolSwapNotAllowed = errors.New("swapping between tokens in the same base pool is not allowed")
	ErrTokenNotRegistered         = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve             = errors.New("invalid reserve")
	ErrInvalidAmountIn            = errors.New("invalid amount in")
	ErrInvalidAmountOut           = errors.New("invalid amount out")
	ErrInvalidSwapFeePercentage   = errors.New("invalid swap fee percentage")
	ErrPoolPaused                 = errors.New("pool is paused")
	ErrMaxTotalInRatio            = errors.New("MAX_TOTAL_IN_RATIO")
	ErrMaxTotalOutRatio           = errors.New("MAX_TOTAL_OUT_RATIO")
	ErrOverflow                   = errors.New("OVERFLOW")
)

var (
	defaultGas = Gas{Swap: 80000}

	_MAX_IN_RATIO  = uint256.NewInt(0.3e18)
	_MAX_OUT_RATIO = uint256.NewInt(0.3e18)
)

type (
	PoolSimulator struct {
		pool.Pool
		basePools map[string]shared.IBasePool
		paused    bool

		swapFeePercentage *uint256.Int
		scalingFactors    []*uint256.Int
		normalizedWeights []*uint256.Int

		vault       string
		poolID      string
		poolTypeVer int

		totalAmountsIn          []*uint256.Int
		scaledMaxTotalAmountsIn []*uint256.Int

		totalAmountsOut          []*uint256.Int
		scaledMaxTotalAmountsOut []*uint256.Int
	}

	Gas struct {
		Swap int64
	}
)

var _ = pool.RegisterFactoryMeta(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))

		totalAmountsIn          = make([]*uint256.Int, len(entityPool.Tokens))
		scaledMaxTotalAmountsIn = make([]*uint256.Int, len(entityPool.Tokens))

		totalAmountsOut          = make([]*uint256.Int, len(entityPool.Tokens))
		scaledMaxTotalAmountsOut = make([]*uint256.Int, len(entityPool.Tokens))
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var basePools = make(map[string]shared.IBasePool, len(staticExtra.BasePools))
	if basePoolMap != nil {
		for basePool := range staticExtra.BasePools {
			if p, ok := basePoolMap[basePool]; ok {
				basePools[basePool] = p.(shared.IBasePool)
			}
		}
	}

	for idx := range entityPool.Tokens {
		tokens[idx] = entityPool.Tokens[idx].Address
		reserves[idx] = bignumber.NewBig10(entityPool.Reserves[idx])
	}

	scaledInitialBalances, err := _upscaleArray(staticExtra.PoolTypeVer, reserves, staticExtra.ScalingFactors)
	if err != nil {
		return nil, err
	}
	for idx := range entityPool.Tokens {
		totalAmountsIn[idx] = number.Zero
		totalAmountsOut[idx] = number.Zero

		maxIn, err := math.FixedPoint.MulDown(scaledInitialBalances[idx], _MAX_IN_RATIO)
		if err != nil {
			return nil, err
		}
		scaledMaxTotalAmountsIn[idx] = maxIn

		maxOut, err := math.FixedPoint.MulDown(scaledInitialBalances[idx], _MAX_OUT_RATIO)
		if err != nil {
			return nil, err
		}
		scaledMaxTotalAmountsOut[idx] = maxOut
	}

	poolInfo := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: uint64(entityPool.BlockNumber),
	}

	p := &PoolSimulator{
		Pool:                     pool.Pool{Info: poolInfo},
		basePools:                basePools,
		paused:                   extra.Paused,
		swapFeePercentage:        extra.SwapFeePercentage,
		scalingFactors:           staticExtra.ScalingFactors,
		normalizedWeights:        staticExtra.NormalizedWeights,
		vault:                    staticExtra.Vault,
		poolID:                   staticExtra.PoolID,
		poolTypeVer:              staticExtra.PoolTypeVer,
		totalAmountsIn:           totalAmountsIn,
		scaledMaxTotalAmountsIn:  scaledMaxTotalAmountsIn,
		totalAmountsOut:          totalAmountsOut,
		scaledMaxTotalAmountsOut: scaledMaxTotalAmountsOut,
	}

	return p, nil
}

func (s *PoolSimulator) GetPoolId() string {
	return s.poolID
}

func (s *PoolSimulator) OnSwap(indexIn, indexOut int, amountIn *uint256.Int) (*uint256.Int, error) {
	balanceIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrOverflow
	}

	balanceOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrOverflow
	}

	scaledBalanceIn, err := _upscale(s.poolTypeVer, balanceIn, s.scalingFactors[indexIn])
	if err != nil {
		return nil, err
	}

	scaledBalanceOut, err := _upscale(s.poolTypeVer, balanceOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, err
	}

	feeAmount, err := math.FixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err := math.FixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}

	if err := s.validateMaxInRatio(indexIn, amountIn); err != nil {
		return nil, err
	}

	scaledAmountIn, err := _upscale(s.poolTypeVer, amountInAfterFee, s.scalingFactors[indexIn])
	if err != nil {
		return nil, err
	}

	scaledAmountOut, err := s._onSwapGivenIn(
		scaledBalanceIn,
		s.normalizedWeights[indexIn],
		scaledBalanceOut,
		s.normalizedWeights[indexOut],
		scaledAmountIn,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err := _downscaleDown(s.poolTypeVer, scaledAmountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (s *PoolSimulator) swapDirect(indexIn, indexOut int, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	amountOut, err := s.OnSwap(indexIn, indexOut, amountIn)
	if err != nil {
		return nil, err
	}

	return s.buildSwapResult(s.Info.Tokens[indexOut], amountOut, nil), nil
}

func (s *PoolSimulator) swapFromBase2Main(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	basePool, err := s.getBasePool(tokenIn)
	if err != nil {
		return nil, err
	}

	bptToken := basePool.GetAddress()

	res, err := basePool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		TokenOut:      bptToken,
	})
	if err != nil {
		return nil, err
	}

	bptAmount, overflow := uint256.FromBig(res.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	bptIndex := s.GetTokenIndex(bptToken)
	indexOut := s.GetTokenIndex(tokenOut)

	amountOut, err := s.OnSwap(bptIndex, indexOut, bptAmount)
	if err != nil {
		return nil, err
	}

	hops := []shared.Hop{
		{
			PoolId:    basePool.GetPoolId(),
			Pool:      basePool.GetAddress(),
			TokenIn:   tokenIn,
			TokenOut:  bptToken,
			AmountIn:  amountIn.ToBig(),
			AmountOut: bptAmount.ToBig(),
		},
		{
			PoolId:    s.GetPoolId(),
			Pool:      s.GetAddress(),
			TokenIn:   bptToken,
			TokenOut:  tokenOut,
			AmountIn:  bptAmount.ToBig(),
			AmountOut: amountOut.ToBig(),
		},
	}

	return s.buildSwapResult(tokenOut, amountOut, hops), nil
}

func (s *PoolSimulator) swapFromMain2Base(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	basePool, err := s.getBasePool(tokenOut)
	if err != nil {
		return nil, err
	}

	bptToken := basePool.GetAddress()

	indexIn := s.GetTokenIndex(tokenIn)
	bptIndex := s.GetTokenIndex(bptToken)

	bptAmount, err := s.OnSwap(indexIn, bptIndex, amountIn)
	if err != nil {
		return nil, err
	}

	res, err := basePool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: bptToken, Amount: bptAmount.ToBig()},
		TokenOut:      tokenOut,
	})
	if err != nil {
		return nil, err
	}

	amountOut, overflow := uint256.FromBig(res.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	hops := []shared.Hop{
		{
			PoolId:    s.poolID,
			Pool:      s.GetAddress(),
			TokenIn:   tokenIn,
			TokenOut:  bptToken,
			AmountIn:  amountIn.ToBig(),
			AmountOut: bptAmount.ToBig(),
		},
		{
			PoolId:    basePool.GetPoolId(),
			Pool:      basePool.GetAddress(),
			TokenIn:   bptToken,
			TokenOut:  tokenOut,
			AmountIn:  bptAmount.ToBig(),
			AmountOut: amountOut.ToBig(),
		},
	}

	return s.buildSwapResult(tokenOut, amountOut, hops), nil
}

func (s *PoolSimulator) swapBetweenBasePools(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	basePoolIn, err := s.getBasePool(tokenIn)
	if err != nil {
		return nil, err
	}

	basePoolOut, err := s.getBasePool(tokenOut)
	if err != nil {
		return nil, err
	}

	bptTokenIn := basePoolIn.GetAddress()
	bptTokenOut := basePoolOut.GetAddress()

	if bptTokenIn == bptTokenOut {
		return nil, ErrSameBasePoolSwapNotAllowed
	}

	res, err := basePoolIn.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		TokenOut:      bptTokenIn,
	})
	if err != nil {
		return nil, err
	}

	bptAmountIn, overflow := uint256.FromBig(res.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	bptInIndex := s.GetTokenIndex(bptTokenIn)
	bptOutIndex := s.GetTokenIndex(bptTokenOut)

	bptAmountOut, err := s.OnSwap(bptInIndex, bptOutIndex, bptAmountIn)
	if err != nil {
		return nil, err
	}

	res, err = basePoolOut.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: bptTokenOut, Amount: bptAmountOut.ToBig()},
		TokenOut:      tokenOut,
	})
	if err != nil {
		return nil, err
	}

	amountOut, overflow := uint256.FromBig(res.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	hops := []shared.Hop{
		{
			PoolId:    basePoolIn.GetPoolId(),
			Pool:      basePoolIn.GetAddress(),
			TokenIn:   tokenIn,
			TokenOut:  bptTokenIn,
			AmountIn:  amountIn.ToBig(),
			AmountOut: bptAmountIn.ToBig(),
		},
		{
			PoolId:    s.poolID,
			Pool:      s.GetAddress(),
			TokenIn:   bptTokenIn,
			TokenOut:  bptTokenOut,
			AmountIn:  bptAmountIn.ToBig(),
			AmountOut: bptAmountOut.ToBig(),
		},
		{
			PoolId:    basePoolOut.GetPoolId(),
			Pool:      basePoolOut.GetAddress(),
			TokenIn:   bptTokenOut,
			TokenOut:  tokenOut,
			AmountIn:  bptAmountOut.ToBig(),
			AmountOut: amountOut.ToBig(),
		},
	}

	return s.buildSwapResult(tokenOut, amountOut, hops), nil
}

func (s *PoolSimulator) buildSwapResult(tokenOut string, amountOut *uint256.Int, hops []shared.Hop) *pool.CalcAmountOutResult {
	var swapInfo shared.SwapInfo
	if hops != nil {
		swapInfo = shared.SwapInfo{
			Hops: hops,
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            defaultGas.Swap,
		SwapInfo:       swapInfo,
	}
}

func (s *PoolSimulator) getBasePool(token string) (shared.IBasePool, error) {
	for _, basePool := range s.basePools {
		index := basePool.GetTokenIndex(token)
		if index >= 0 {
			return basePool, nil
		}
	}

	return nil, ErrTokenNotRegistered
}

// https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F7#L32
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)

	if indexIn >= 0 && indexOut >= 0 {
		return s.swapDirect(indexIn, indexOut, amountIn)
	}

	if indexIn < 0 && indexOut >= 0 {
		return s.swapFromBase2Main(tokenAmountIn.Token, tokenOut, amountIn)
	}

	if indexIn >= 0 && indexOut < 0 {
		return s.swapFromMain2Base(tokenAmountIn.Token, tokenOut, amountIn)
	}

	return s.swapBetweenBasePools(tokenAmountIn.Token, tokenOut, amountIn)
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountOut := params.TokenAmountOut
	tokenIn := params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)

	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if err := s.validateMaxOutRatio(indexOut, amountOut); err != nil {
		return nil, err
	}

	scalingFactorTokenIn := s.scalingFactors[indexIn]
	scalingFactorTokenOut := s.scalingFactors[indexOut]
	normalizedWeightIn := s.normalizedWeights[indexIn]
	normalizedWeightOut := s.normalizedWeights[indexOut]

	balanceTokenIn, err := _upscale(s.poolTypeVer, reserveIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}
	balanceTokenOut, err := _upscale(s.poolTypeVer, reserveOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	upScaledAmountOut, err := _upscale(s.poolTypeVer, amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	upScaledAmountIn, err := s._onSwapGivenOut(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountOut,
	)
	if err != nil {
		return nil, err
	}

	amountIn, err := _downscaleUp(s.poolTypeVer, upScaledAmountIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err := s._addSwapFeeAmount(amountIn)
	if err != nil {
		return nil, err
	}

	feeAmount, err := math.FixedPoint.Sub(amountInAfterFee, amountIn)
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
			Amount: feeAmount.ToBig(),
		},
		Gas: defaultGas.Swap,
	}, nil
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F1#L165
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F3#L117
func (s *PoolSimulator) _onSwapGivenIn(
	balanceTokenIn *uint256.Int,
	normalizedWeightIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	normalizedWeightOut *uint256.Int,
	upScaledAmountIn *uint256.Int,
) (*uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer1 {
		return math.WeightedMath.CalcOutGivenInV1(
			balanceTokenIn,
			normalizedWeightIn,
			balanceTokenOut,
			normalizedWeightOut,
			upScaledAmountIn,
		)
	}

	return math.WeightedMath.CalcOutGivenIn(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountIn,
	)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F1#L182
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F3#L132
func (s *PoolSimulator) _onSwapGivenOut(
	balanceTokenIn *uint256.Int,
	normalizedWeightIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	normalizedWeightOut *uint256.Int,
	upScaledAmountOut *uint256.Int,
) (*uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer1 {
		return math.WeightedMath.CalcInGivenOutV1(
			balanceTokenIn,
			normalizedWeightIn,
			balanceTokenOut,
			normalizedWeightOut,
			upScaledAmountOut,
		)
	}

	return math.WeightedMath.CalcInGivenOut(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountOut,
	)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L454
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L619
func (s *PoolSimulator) _addSwapFeeAmount(amount *uint256.Int) (*uint256.Int, error) {
	// This returns amount + fee amount, so we round up (favoring a higher fee amount).
	return math.FixedPoint.DivUp(amount, math.FixedPoint.Complement(s.swapFeePercentage))
}

func (s *PoolSimulator) validateMaxInRatio(tokenIndex int, amountIn *uint256.Int) error {
	sum := new(uint256.Int).Add(s.totalAmountsIn[tokenIndex], amountIn)
	upscaledSum, err := _upscale(s.poolTypeVer, sum, s.scalingFactors[tokenIndex])
	if err != nil {
		return err
	}

	if upscaledSum.Gt(s.scaledMaxTotalAmountsIn[tokenIndex]) {
		return ErrMaxTotalInRatio
	}

	return nil
}

func (s *PoolSimulator) validateMaxOutRatio(tokenIndex int, amountOut *uint256.Int) error {
	sum := new(uint256.Int).Add(s.totalAmountsOut[tokenIndex], amountOut)
	upscaledSum, err := _upscale(s.poolTypeVer, sum, s.scalingFactors[tokenIndex])
	if err != nil {
		return err
	}

	if upscaledSum.Gt(s.scaledMaxTotalAmountsOut[tokenIndex]) {
		return ErrMaxTotalOutRatio
	}

	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.SwapInfo == nil {
		s.updateBalance(params.TokenAmountIn.Token, params.TokenAmountOut.Token,
			params.TokenAmountIn.Amount, params.TokenAmountOut.Amount)

		return
	}

	if swapInfo, ok := params.SwapInfo.(shared.SwapInfo); ok {
		for _, hop := range swapInfo.Hops {
			if basePool, ok := s.basePools[hop.Pool]; ok {
				basePool.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  hop.TokenIn,
						Amount: hop.AmountIn,
					},
					TokenAmountOut: pool.TokenAmount{
						Token:  hop.TokenOut,
						Amount: hop.AmountOut,
					},
				})
			} else {
				s.updateBalance(hop.TokenIn, hop.TokenOut, hop.AmountIn, hop.AmountOut)
			}
		}
	}
}

func (s *PoolSimulator) updateBalance(tokenIn, tokenOut string, amountIn, amountOut *big.Int) {
	for idx, token := range s.Info.Tokens {
		if token == tokenIn {
			s.Info.Reserves[idx] = new(big.Int).Add(
				s.Info.Reserves[idx],
				amountIn,
			)

			s.totalAmountsIn[idx] = new(uint256.Int).Add(
				s.totalAmountsIn[idx],
				uint256.MustFromBig(amountIn),
			)
		}

		if token == tokenOut {
			s.Info.Reserves[idx] = new(big.Int).Sub(
				s.Info.Reserves[idx],
				amountOut,
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

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F27#L529
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F13#L681
func _upscale(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.Mul(amount, scalingFactor)
	}

	return math.FixedPoint.MulDown(amount, scalingFactor)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L547
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L706
func _downscaleDown(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.DivDown(amount, scalingFactor)
	}

	return math.FixedPoint.DivDown(amount, scalingFactor)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L565
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L727
func _downscaleUp(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.DivUp(amount, scalingFactor)
	}

	return math.FixedPoint.DivUp(amount, scalingFactor)
}

func _upscaleArray(poolTypeVer int, balances []*big.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(balances))
	for i, balance := range balances {
		b, overflow := uint256.FromBig(balance)
		if overflow {
			return nil, ErrOverflow
		}

		upscaledI, err := _upscale(poolTypeVer, b, scalingFactors[i])
		if err != nil {
			return nil, err
		}
		upscaled[i] = upscaledI
	}
	return upscaled, nil
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	return t.CanSwapTo(address)
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	result := make(map[string]struct{})
	var tokenIndex = t.GetTokenIndex(address)

	if tokenIndex < 0 {
		found := false // Flag to check if any base pool contains the token

		for _, basePool := range t.basePools {
			if basePool.GetTokenIndex(address) >= 0 {
				found = true

				for _, token := range basePool.CanSwapTo(address) {
					result[token] = struct{}{}
				}
			} else {
				for _, underlyingToken := range basePool.GetTokens() {
					if underlyingToken != address {
						result[underlyingToken] = struct{}{}
					}
				}
			}
		}

		if !found {
			return []string{}
		}

		// Add tokens from main pool
		for _, poolToken := range t.GetTokens() {
			result[poolToken] = struct{}{}
		}
	} else {

		// Add tokens from main pool except itself
		for _, poolToken := range t.GetTokens() {
			if poolToken != address {
				result[poolToken] = struct{}{}
			}
		}

		for _, basePool := range t.basePools {
			for _, underlyingToken := range basePool.GetTokens() {
				if underlyingToken != address {
					result[underlyingToken] = struct{}{}
				}
			}
		}
	}

	return lo.Keys(result)
}

func (t *PoolSimulator) GetTokens() []string {
	tokenSet := make(map[string]struct{})

	for _, basePool := range t.basePools {
		for _, token := range basePool.GetTokens() {
			tokenSet[token] = struct{}{}
		}
	}

	for _, token := range t.GetInfo().Tokens {
		tokenSet[token] = struct{}{}
	}

	return lo.Keys(tokenSet)
}

func (t *PoolSimulator) GetBasePool() pool.IPoolSimulator {
	for _, basePool := range t.basePools {
		return basePool
	}

	return nil
}

func (t *PoolSimulator) SetBasePool(basePool pool.IPoolSimulator) {
	if basePool, ok := basePool.(shared.IBasePool); ok {
		t.basePools[basePool.GetAddress()] = basePool
	}
}
