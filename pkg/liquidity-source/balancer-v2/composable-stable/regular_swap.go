//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple regularSimulator
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256

package composablestable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type regularSimulator struct {
	poolpkg.Pool

	bptIndex          int
	scalingFactors    []*uint256.Int
	amp               *uint256.Int
	swapFeePercentage *uint256.Int
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L184
// It calls `super._swapGivenIn`, which is the code below:
// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F11#L49
func (s *regularSimulator) _swapGivenIn(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, *poolpkg.TokenAmount, *SwapInfo, error) {
	feeAmount, err := math.FixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, nil, nil, err
	}
	amountInAfterFee, err := math.FixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, nil, nil, err
	}

	balances, err = _upscaleArray(balances, s.scalingFactors)
	if err != nil {
		return nil, nil, nil, err
	}

	upScaledAmountInAfterFee, err := _upscale(amountInAfterFee, s.scalingFactors[indexIn])
	if err != nil {
		return nil, nil, nil, err
	}

	upscaledAmountOut, err := s._onSwapGivenIn(upScaledAmountInAfterFee, balances, indexIn, indexOut)
	if err != nil {
		return nil, nil, nil, err
	}

	amountOut, err := _downscaleDown(upscaledAmountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	fee := poolpkg.TokenAmount{
		Token:  s.Info.Tokens[indexIn],
		Amount: feeAmount.ToBig(),
	}

	return amountOut, &fee, &SwapInfo{}, nil
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L215
// It calls `super._swapGivenOut`, which is the code below:
// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F11#L68
func (s *regularSimulator) _swapGivenOut(
	amountOut *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, *poolpkg.TokenAmount, *SwapInfo, error) {
	balances, err := _upscaleArray(balances, s.scalingFactors)
	if err != nil {
		return nil, nil, nil, err
	}

	upScaledAmountOut, err := _upscale(amountOut, s.scalingFactors[indexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	upscaledAmountIn, err := s._onSwapGivenOut(upScaledAmountOut, balances, indexIn, indexOut)
	if err != nil {
		return nil, nil, nil, err
	}

	amountIn, err := _downscaleUp(upscaledAmountIn, s.scalingFactors[indexIn])
	if err != nil {
		return nil, nil, nil, err
	}

	// Fees are added after scaling happens, to reduce the complexity of the rounding direction analysis.
	amountInAfterFee, err := s._addSwapFeeAmount(amountIn)
	if err != nil {
		return nil, nil, nil, err
	}

	feeAmount, err := math.FixedPoint.Sub(amountInAfterFee, amountIn)
	if err != nil {
		return nil, nil, nil, err
	}

	fee := poolpkg.TokenAmount{
		Token:  s.Info.Tokens[indexIn],
		Amount: feeAmount.ToBig(),
	}

	return amountIn, &fee, &SwapInfo{}, nil
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L229
func (s *regularSimulator) _onSwapGivenIn(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	return s._onRegularSwap(
		true, // given in
		amountIn,
		balances,
		indexIn,
		indexOut,
	)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L250
func (s *regularSimulator) _onSwapGivenOut(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	return s._onRegularSwap(
		false, // given out
		amountIn,
		balances,
		indexIn,
		indexOut,
	)
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F1#L270
func (s *regularSimulator) _onRegularSwap(
	isGivenIn bool,
	amountGiven *uint256.Int,
	registeredBalances []*uint256.Int,
	registeredIndexIn int,
	registeredIndexOut int,
) (*uint256.Int, error) {
	balances := _dropBptItem(registeredBalances, s.bptIndex)
	indexIn, indexOut := _skipBptIndex(registeredIndexIn, s.bptIndex), _skipBptIndex(registeredIndexOut, s.bptIndex)

	invariant, err := math.StableMath.CalculateInvariantV2(s.amp, balances)
	if err != nil {
		return nil, err
	}

	if isGivenIn {
		return math.StableMath.CalcOutGivenIn(
			invariant,
			s.amp,
			amountGiven,
			balances,
			indexIn,
			indexOut,
		)
	}

	return math.StableMath.CalcInGivenOut(
		invariant,
		s.amp,
		amountGiven,
		balances,
		indexIn,
		indexOut,
	)
}

func (s *regularSimulator) updateBalance(params poolpkg.UpdateBalanceParams) {
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

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F22#L609
func (s *regularSimulator) _addSwapFeeAmount(amount *uint256.Int) (*uint256.Int, error) {
	// This returns amount + fee amount, so we round up (favoring a higher fee amount).
	return math.FixedPoint.DivUp(amount, math.FixedPoint.Complement(s.swapFeePercentage))
}
