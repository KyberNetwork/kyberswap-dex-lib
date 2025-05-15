package composablestable

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	paused                 bool
	canNotUpdateTokenRates bool

	regularSimulator *regularSimulator
	bptSimulator     *bptSimulator

	vault       string
	poolID      string
	poolTypeVer int
}

var _ shared.IBasePool = (*PoolSimulator)(nil)

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

	pool := pool.Pool{
		Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
	}

	regularSimulator := regularSimulator{
		Pool:              pool,
		bptIndex:          staticExtra.BptIndex,
		scalingFactors:    extra.ScalingFactors,
		amp:               extra.Amp,
		swapFeePercentage: extra.SwapFeePercentage,
	}

	bptSimulator := bptSimulator{
		Pool:                            pool,
		bptIndex:                        staticExtra.BptIndex,
		bptTotalSupply:                  extra.BptTotalSupply,
		amp:                             extra.Amp,
		scalingFactors:                  extra.ScalingFactors,
		lastJoinExit:                    extra.LastJoinExit,
		rateProviders:                   extra.RateProviders,
		tokenRateCaches:                 extra.TokenRateCaches,
		swapFeePercentage:               extra.SwapFeePercentage,
		protocolFeePercentageCache:      extra.ProtocolFeePercentageCache,
		tokenExemptFromYieldProtocolFee: extra.IsTokenExemptFromYieldProtocolFee,
		exemptFromYieldProtocolFee:      extra.IsExemptFromYieldProtocolFee,
		inRecoveryMode:                  extra.InRecoveryMode,

		poolTypeVer: staticExtra.PoolTypeVer,
	}

	return &PoolSimulator{
		Pool:                   pool,
		paused:                 extra.Paused,
		canNotUpdateTokenRates: extra.CanNotUpdateTokenRates,
		regularSimulator:       &regularSimulator,
		bptSimulator:           &bptSimulator,
		vault:                  staticExtra.Vault,
		poolID:                 staticExtra.PoolID,
		poolTypeVer:            staticExtra.PoolTypeVer,
	}, nil
}

func (s *PoolSimulator) GetPoolId() string {
	return s.poolID
}

func (s *PoolSimulator) OnJoin(tokenIn string, amountIn *uint256.Int) (*uint256.Int, error) {
	return number.Zero, nil
}

func (s *PoolSimulator) OnExit(tokenOut string, amountIn *uint256.Int) (*uint256.Int, error) {
	return number.Zero, nil
}

func (s *PoolSimulator) OnSwap(tokenIn, tokenOut string, amountIn *uint256.Int) (*uint256.Int, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if s.canNotUpdateTokenRates {
		return nil, ErrBeforeSwapJoinExit
	}

	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrUnknownToken
	}

	balances := make([]*uint256.Int, len(s.Info.Reserves))
	for i, reserve := range s.Info.Reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrOverflow
		}
		balances[i] = r
	}

	var (
		amountOut *uint256.Int
		err       error
	)
	if tokenIn == s.Info.Address || tokenOut == s.Info.Address {
		amountOut, _, _, err = s.bptSimulator._swapWithBpt(true, amountIn, balances, indexIn, indexOut)
	} else {
		amountOut, _, _, err = s.regularSimulator._swapGivenIn(amountIn, balances, indexIn, indexOut)
	}

	return amountOut, err
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if s.canNotUpdateTokenRates {
		return nil, ErrBeforeSwapJoinExit
	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut

	indexIn := s.GetTokenIndex(tokenAmountIn.Token)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrUnknownToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	balances := make([]*uint256.Int, len(s.Info.Reserves))
	for i, reserve := range s.Info.Reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrOverflow
		}
		balances[i] = r
	}

	var (
		amountOut *uint256.Int
		fee       *pool.TokenAmount
		swapInfo  *SwapInfo
		err       error
	)
	if tokenAmountIn.Token == s.Info.Address || tokenOut == s.Info.Address {
		amountOut, fee, swapInfo, err = s.bptSimulator._swapWithBpt(true, amountIn, balances, indexIn, indexOut)
	} else {
		amountOut, fee, swapInfo, err = s.regularSimulator._swapGivenIn(amountIn, balances, indexIn, indexOut)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee:      fee,
		Gas:      DefaultGas.Swap,
		SwapInfo: swapInfo,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if s.canNotUpdateTokenRates {
		return nil, ErrBeforeSwapJoinExit
	}

	tokenAmountOut := params.TokenAmountOut
	tokenIn := params.TokenIn

	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrUnknownToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	balances := make([]*uint256.Int, len(s.Info.Reserves))
	for i, reserve := range s.Info.Reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrOverflow
		}
		balances[i] = r
	}

	var (
		amountOut *uint256.Int
		fee       *pool.TokenAmount
		swapInfo  *SwapInfo
		err       error
	)
	if tokenAmountOut.Token == s.Info.Address || tokenIn == s.Info.Address {
		amountOut, fee, swapInfo, err = s.bptSimulator._swapWithBpt(false, amountIn, balances, indexIn, indexOut)
	} else {
		amountOut, fee, swapInfo, err = s.regularSimulator._swapGivenOut(amountIn, balances, indexIn, indexOut)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountOut.ToBig(),
		},
		Fee:      fee,
		Gas:      DefaultGas.Swap,
		SwapInfo: swapInfo,
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
	if params.TokenAmountIn.Token == s.Info.Address || params.TokenAmountOut.Token == s.Info.Address {
		s.bptSimulator.updateBalance(params)
		return
	}

	s.regularSimulator.updateBalance(params)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F22#L696
/**
 * @dev Reverses the `scalingFactor` applied to `amount`, resulting in a smaller or equal value depending on
 * whether it needed scaling or not. The result is rounded down.
 */
func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F22#L717
/**
 * @dev Reverses the `scalingFactor` applied to `amount`, resulting in a smaller or equal value depending on
 * whether it needed scaling or not. The result is rounded up.
 */
func _downscaleUp(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivUp(amount, scalingFactor)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F22#L683
/**
 * @dev Same as `_upscale`, but for an entire array. This function does not return anything, but instead *mutates*
 * the `amounts` array.
 */
func _upscaleArray(balances []*uint256.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(balances))
	for i, balance := range balances {
		upscaledI, err := _upscale(balance, scalingFactors[i])
		if err != nil {
			return nil, err
		}
		upscaled[i] = upscaledI
	}
	return upscaled, nil
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F22#L671
/**
 * @dev Applies `scalingFactor` to `amount`, resulting in a larger or equal value depending on whether it needed
 * scaling or not.
 */
func _upscale(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.MulDown(amount, scalingFactor)
}

func _dropBptItem(amounts []*uint256.Int, bptIndex int) []*uint256.Int {
	amountsWithoutBpt := make([]*uint256.Int, len(amounts)-1)

	for i := 0; i < len(amountsWithoutBpt); i++ {
		if i < bptIndex {
			amountsWithoutBpt[i] = amounts[i]
			continue
		}
		amountsWithoutBpt[i] = amounts[i+1]
	}

	return amountsWithoutBpt
}

func _skipBptIndex(index int, bptIndex int) int {
	if index < bptIndex {
		return index
	}
	return index - 1
}
