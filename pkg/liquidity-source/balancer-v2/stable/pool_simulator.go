package stable

import (
	"errors"
	"log"
	"math/big"

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
	ErrSameBasePoolSwapNotAllowed = errors.New("swapping between tokens in the same base pool is not allowed")
	ErrInvalidSwapFeePercentage   = errors.New("invalid swap fee percentage")
	ErrPoolPaused                 = errors.New("pool is paused")
	ErrInvalidAmp                 = errors.New("invalid amp")
	ErrNotTwoTokens               = errors.New("not two tokens")
)

type PoolSimulator struct {
	pool.Pool
	basePools map[string]shared.IBasePool

	paused bool

	swapFeePercentage *uint256.Int
	amp               *uint256.Int

	scalingFactors []*uint256.Int

	vault    string
	poolID   string
	poolSpec uint8

	poolType    string
	poolTypeVer int
}

var _ = pool.RegisterFactoryMeta(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
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
		basePools:         basePools,
		paused:            extra.Paused,
		swapFeePercentage: extra.SwapFeePercentage,
		amp:               extra.Amp,
		scalingFactors:    extra.ScalingFactors,
		vault:             staticExtra.Vault,
		poolID:            staticExtra.PoolID,
		poolSpec:          staticExtra.PoolSpecialization,
		poolType:          staticExtra.PoolType,
		poolTypeVer:       staticExtra.PoolTypeVer,
	}, nil
}

func (s *PoolSimulator) GetPoolId() string {
	return s.poolID
}

func (s *PoolSimulator) OnSwap(indexIn, indexOut int, amountIn *uint256.Int) (*uint256.Int, error) {
	log.Fatalln("")
	feeAmount, err := math.FixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err := math.FixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}

	scaledAmountIn, err := _upscale(amountInAfterFee, s.scalingFactors[indexIn])
	if err != nil {
		return nil, err
	}

	scaledBalances, err := _upscaleArray(s.Info.Reserves, s.scalingFactors)
	if err != nil {
		return nil, err
	}

	invariant, err := calculateInvariant(s.poolType, s.poolTypeVer, s.amp, scaledBalances)
	if err != nil {
		return nil, err
	}

	amountOut, err := math.StableMath.CalcOutGivenIn(
		invariant,
		s.amp,
		scaledAmountIn,
		scaledBalances,
		indexIn,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	// amountOut tokens are exiting the Pool, so we round down.
	amountOut, err = _downscaleDown(amountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (s *PoolSimulator) getBasePool(token string) (shared.IBasePool, error) {
	for _, basePool := range s.basePools {
		index := basePool.GetTokenIndex(token)
		if index > 0 {
			return basePool, nil
		}
	}

	return nil, ErrTokenNotRegistered
}

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F6#L46
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if s.poolSpec != poolSpecializationGeneral && len(s.Info.Tokens) != 2 {
		return nil, ErrNotTwoTokens
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

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F6#L65
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	// NOTE: if pool specialization is not "General", then the pool must have 2 tokens
	// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F1#L130
	if s.poolSpec != poolSpecializationGeneral && len(s.Info.Tokens) != 2 {
		return nil, ErrNotTwoTokens
	}

	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	amountOut, err := _upscale(amountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, err
	}

	balances, err := _upscaleArray(s.Info.Reserves, s.scalingFactors)
	if err != nil {
		return nil, err
	}

	invariant, err := calculateInvariant(s.poolType, s.poolTypeVer, s.amp, balances)
	if err != nil {
		return nil, err
	}

	amountIn, err := math.StableMath.CalcInGivenOut(
		invariant,
		s.amp,
		amountOut,
		balances,
		indexIn,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	// amountIn tokens are entering the Pool, so we round up.
	amountIn, err = _downscaleUp(amountIn, s.scalingFactors[indexIn])
	if err != nil {
		return nil, err
	}

	// Fees are added after scaling happens, to reduce the complexity of the rounding direction analysis.
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

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vault,
		PoolID:        s.poolID,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
	}
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
		}

		if token == tokenOut {
			s.Info.Reserves[idx] = new(big.Int).Sub(
				s.Info.Reserves[idx],
				amountOut,
			)
		}
	}
}

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F13#L490
// The exact implementation is just `return amount.divUp(FixedPoint.ONE.sub(_swapFeePercentage));`
// But we use Complement to avoid negative value.
func (s *PoolSimulator) _addSwapFeeAmount(amount *uint256.Int) (*uint256.Int, error) {
	// This returns amount + fee amount, so we round up (favoring a higher fee amount).
	return math.FixedPoint.DivUp(amount, math.FixedPoint.Complement(s.swapFeePercentage))
}

// MetaStable: https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F30#L49
//
// Stable Version 1: https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F8#L49
//
// Stable Version 2: https://etherscan.io/address/0x13f2f70a951fb99d48ede6e25b0bdf06914db33f#code#F5#L57
func calculateInvariant(
	poolType string,
	poolTypeVer int,
	amp *uint256.Int,
	balances []*uint256.Int,
) (*uint256.Int, error) {
	if poolType == poolTypeMetaStable {
		return math.StableMath.CalculateInvariantV1(amp, balances, true)
	}

	if poolTypeVer == poolTypeVer1 {
		return math.StableMath.CalculateInvariantV1(amp, balances, true)
	}

	return math.StableMath.CalculateInvariantV2(amp, balances)
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F31#L530
func _upscaleArray(reserves []*big.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(reserves))
	for i, reserve := range reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrInvalidReserve
		}

		upscaledI, err := _upscale(r, scalingFactors[i])
		if err != nil {
			return nil, err
		}
		upscaled[i] = upscaledI
	}
	return upscaled, nil
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F31#L518
func _upscale(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.MulDown(amount, scalingFactor)
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F32#L540
func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F32#L558
func _downscaleUp(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivUp(amount, scalingFactor)
}

func (t *PoolSimulator) GetTokens() []string {
	var p []string
	for _, v := range t.basePools {
		p = append(p, v.GetTokens()...)
	}

	return p
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

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
