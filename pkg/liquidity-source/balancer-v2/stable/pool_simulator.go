package stable

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
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
	ErrBatchSwapDisabled          = errors.New("batch swap is disabled")
)

type PoolSimulator struct {
	pool.Pool
	basePools map[string]shared.IBasePool

	paused bool

	protocolSwapFeePercentage *uint256.Int
	swapFeePercentage         *uint256.Int
	amp                       *uint256.Int
	totalSupply               *uint256.Int

	scalingFactors []*uint256.Int

	vault    string
	poolID   string
	poolSpec uint8

	poolType    string
	poolTypeVer int

	batchSwapEnabled bool
}

var _ shared.IBasePool = (*PoolSimulator)(nil)

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
		batchSwapEnabled:  staticExtra.BatchSwapEnabled,
	}, nil
}

func (s *PoolSimulator) GetPoolId() string {
	return s.poolID
}

func (s *PoolSimulator) OnJoin(tokenIn string, amountIn *uint256.Int) (*uint256.Int, error) {
	indexIn := s.GetTokenIndex(tokenIn)

	scaledBalances, err := _upscaleArray(s.GetReserves(), s.scalingFactors)
	if err != nil {
		return nil, err
	}

	var scaledAmountsIn = make([]*uint256.Int, len(s.Info.Tokens))

	for i := range s.Info.Tokens {
		if i == indexIn {
			scaledAmountsIn[i], err = _upscale(amountIn, s.scalingFactors[i])
			if err != nil {
				return nil, err
			}
		} else {
			scaledAmountsIn[i] = number.Zero
		}
	}

	invariant, err := calculateInvariant(s.poolType, s.poolTypeVer, s.amp, scaledBalances)
	if err != nil {
		return nil, err
	}

	err = chargeDueProtocolFee(scaledBalances, s.amp, invariant, s.protocolSwapFeePercentage)
	if err != nil {
		return nil, err
	}

	bptAmountOut, err := math.StableMath.CalcBptOutGivenExactTokensIn(s.amp,
		scaledBalances, scaledAmountsIn, s.totalSupply, invariant, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}

	return bptAmountOut, nil
}

func (s *PoolSimulator) OnExit(tokenOut string, bptAmountIn *uint256.Int) (*uint256.Int, error) {
	indexOut := s.GetTokenIndex(tokenOut)

	scaledBalances, err := _upscaleArray(s.GetReserves(), s.scalingFactors)
	if err != nil {
		return nil, err
	}

	invariant, err := calculateInvariant(s.poolType, s.poolTypeVer, s.amp, scaledBalances)
	if err != nil {
		return nil, err
	}

	err = chargeDueProtocolFee(scaledBalances, s.amp, invariant, s.protocolSwapFeePercentage)
	if err != nil {
		return nil, err
	}

	amountOut, err := math.StableMath.CalcTokenOutGivenExactBptIn(s.amp,
		scaledBalances, indexOut, bptAmountIn, s.totalSupply, invariant, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}

	amountOut, err = _downscaleDown(amountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (s *PoolSimulator) OnSwap(tokenIn, tokenOut string, amountIn *uint256.Int) (*uint256.Int, error) {
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)

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
		if index >= 0 {
			return basePool, nil
		}
	}

	return nil, ErrTokenNotRegistered
}

func chargeDueProtocolFee(
	balances []*uint256.Int,
	amplificationParameter, invariant,
	swapFeePercentage *uint256.Int,
) error {
	if swapFeePercentage.IsZero() {
		return nil
	}

	var chosenTokenIndex = 0
	maxBalance := balances[0]
	for i := 1; i < len(balances); i++ {
		if balances[i].Gt(maxBalance) {
			chosenTokenIndex = i
			maxBalance = balances[i]
		}
	}

	dueProtocolFeeAmount, err := math.StableMath.CalcDueTokenProtocolSwapFeeAmount(amplificationParameter, balances,
		invariant, chosenTokenIndex, swapFeePercentage)
	if err != nil {
		return err
	}

	balances[chosenTokenIndex].Sub(balances[chosenTokenIndex], dueProtocolFeeAmount)

	return nil
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

	if (indexIn < 0 || indexOut < 0) && !s.batchSwapEnabled {
		return nil, ErrBatchSwapDisabled
	}

	if indexIn >= 0 && indexOut >= 0 {
		return s.swapDirect(tokenAmountIn.Token, tokenOut, amountIn)
	}

	if indexIn < 0 && indexOut >= 0 {
		return s.swapFromBase2Main(tokenAmountIn.Token, tokenOut, amountIn)
	}

	if indexIn >= 0 && indexOut < 0 {
		return s.swapFromMain2Base(tokenAmountIn.Token, tokenOut, amountIn)
	}

	return s.swapBetweenBasePools(tokenAmountIn.Token, tokenOut, amountIn)
}

func (s *PoolSimulator) swapDirect(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	amountOut, err := s.OnSwap(tokenIn, tokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	return s.buildSwapResult(tokenOut, amountOut, nil), nil
}

func (s *PoolSimulator) swapFromBase2Main(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	basePool, err := s.getBasePool(tokenIn)
	if err != nil {
		return nil, err
	}

	bptToken := basePool.GetAddress()

	var (
		hops = make([]shared.Hop, 0, 2)

		joinIndex, bptAmount *uint256.Int
	)

	switch basePool.GetType() {
	case composablestable.DexType:
		bptAmount, err = basePool.OnSwap(tokenIn, bptToken, amountIn)
	default:
		bptAmount, err = basePool.OnJoin(tokenIn, amountIn)
		joinIndex = shared.PackJoinExitIndex(shared.PoolJoin, len(hops))
	}
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:        basePool.GetPoolId(),
		Pool:          basePool.GetAddress(),
		TokenIn:       tokenIn,
		TokenOut:      bptToken,
		AmountIn:      amountIn,
		AmountOut:     bptAmount,
		JoinExitIndex: joinIndex,
	})

	amountOut, err := s.OnSwap(bptToken, tokenOut, bptAmount)
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:    s.GetPoolId(),
		Pool:      s.GetAddress(),
		TokenIn:   bptToken,
		TokenOut:  tokenOut,
		AmountIn:  bptAmount,
		AmountOut: amountOut,
	})

	return s.buildSwapResult(tokenOut, amountOut, hops), nil
}

func (s *PoolSimulator) swapFromMain2Base(tokenIn, tokenOut string, amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	basePool, err := s.getBasePool(tokenOut)
	if err != nil {
		return nil, err
	}

	bptToken := basePool.GetAddress()

	var hops = make([]shared.Hop, 0, 2)

	bptAmount, err := s.OnSwap(tokenIn, bptToken, amountIn)
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:    s.poolID,
		Pool:      s.GetAddress(),
		TokenIn:   tokenIn,
		TokenOut:  bptToken,
		AmountIn:  amountIn,
		AmountOut: bptAmount,
	})

	var (
		exitIndex, amountOut *uint256.Int
	)

	switch basePool.GetType() {
	case composablestable.DexType:
		amountOut, err = basePool.OnSwap(bptToken, tokenOut, bptAmount)
	default:
		amountOut, err = basePool.OnExit(tokenOut, bptAmount)
		exitIndex = shared.PackJoinExitIndex(shared.PoolExit, len(hops))
	}
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:        basePool.GetPoolId(),
		Pool:          basePool.GetAddress(),
		TokenIn:       bptToken,
		TokenOut:      tokenOut,
		AmountIn:      bptAmount,
		AmountOut:     amountOut,
		JoinExitIndex: exitIndex,
	})

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

	var (
		hops = make([]shared.Hop, 0, 3)

		joinIndex, bptAmountIn *uint256.Int
	)

	switch basePoolIn.GetType() {
	case composablestable.DexType:
		bptAmountIn, err = basePoolIn.OnSwap(tokenIn, bptTokenIn, amountIn)
	default:
		bptAmountIn, err = basePoolIn.OnJoin(tokenIn, amountIn)
		joinIndex = shared.PackJoinExitIndex(shared.PoolJoin, len(hops))
	}
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:        basePoolIn.GetPoolId(),
		Pool:          basePoolIn.GetAddress(),
		TokenIn:       tokenIn,
		TokenOut:      bptTokenIn,
		AmountIn:      amountIn,
		AmountOut:     bptAmountIn,
		JoinExitIndex: joinIndex,
	})

	bptAmountOut, err := s.OnSwap(bptTokenIn, bptTokenOut, bptAmountIn)
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:    s.poolID,
		Pool:      s.GetAddress(),
		TokenIn:   bptTokenIn,
		TokenOut:  bptTokenOut,
		AmountIn:  bptAmountIn,
		AmountOut: bptAmountOut,
	})

	var (
		exitIndex, amountOut *uint256.Int
	)

	switch basePoolOut.GetType() {
	case composablestable.DexType:
		amountOut, err = basePoolOut.OnSwap(bptTokenOut, tokenOut, bptAmountOut)
	default:
		amountOut, err = basePoolOut.OnExit(tokenOut, bptAmountOut)
		exitIndex = shared.PackJoinExitIndex(shared.PoolExit, len(hops))
	}
	if err != nil {
		return nil, err
	}

	hops = append(hops, shared.Hop{
		PoolId:        basePoolOut.GetPoolId(),
		Pool:          basePoolOut.GetAddress(),
		TokenIn:       bptTokenOut,
		TokenOut:      tokenOut,
		AmountIn:      bptAmountOut,
		AmountOut:     amountOut,
		JoinExitIndex: exitIndex,
	})

	return s.buildSwapResult(tokenOut, amountOut, hops), nil
}

func (s *PoolSimulator) buildSwapResult(tokenOut string, amountOut *uint256.Int, hops []shared.Hop) *pool.CalcAmountOutResult {
	var (
		swapInfo     shared.SwapInfo
		estimatedGas = defaultGas.Swap
	)

	if hops != nil {
		swapInfo = shared.SwapInfo{
			Hops: hops,
		}

		for _, hop := range hops {
			if hop.JoinExitIndex != nil {
				estimatedGas += shared.JoinExitGasUsage
			} else {
				estimatedGas += defaultGas.Swap
			}
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            estimatedGas,
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
			amountIn := hop.AmountIn.ToBig()
			amountOut := hop.AmountOut.ToBig()

			if basePool, ok := s.basePools[hop.Pool]; ok {
				basePool.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  hop.TokenIn,
						Amount: amountIn,
					},
					TokenAmountOut: pool.TokenAmount{
						Token:  hop.TokenOut,
						Amount: amountOut,
					},
				})
			} else {
				s.updateBalance(hop.TokenIn, hop.TokenOut, amountIn, amountOut)
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
	if poolType == poolTypeMetaStable || poolTypeVer == poolTypeVer1 {
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
		for _, poolToken := range t.Info.Tokens {
			result[poolToken] = struct{}{}
		}
	} else {
		// Add tokens from main pool except itself
		for _, poolToken := range t.Info.Tokens {
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

func (t *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	var result = make([]pool.IPoolSimulator, 0, len(t.basePools))
	for _, basePool := range t.basePools {
		result = append(result, basePool)
	}

	return result
}

func (t *PoolSimulator) SetBasePool(basePool pool.IPoolSimulator) {
	if basePool, ok := basePool.(shared.IBasePool); ok {
		t.basePools[basePool.GetAddress()] = basePool
	}
}
