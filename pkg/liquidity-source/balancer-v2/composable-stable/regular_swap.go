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

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F10#L49
func (s *regularSimulator) swap(
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

func (s *regularSimulator) _onSwapGivenIn(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	return s._onRegularSwap(amountIn, balances, indexIn, indexOut)
}

func (s *regularSimulator) _onRegularSwap(
	amountIn *uint256.Int,
	registeredBalances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	balances := _dropBptItem(registeredBalances, s.bptIndex)
	indexIn, indexOut = _skipBptIndex(indexIn, s.bptIndex), _skipBptIndex(indexOut, s.bptIndex)

	invariant, err := math.StableMath.CalculateInvariantV2(s.amp, balances)
	if err != nil {
		return nil, err
	}

	return math.StableMath.CalcOutGivenIn(
		invariant,
		s.amp,
		amountIn,
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
