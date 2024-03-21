package composablestable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type RegularSimulator struct {
	poolpkg.Pool

	BptIndex          int
	ScalingFactors    []*uint256.Int
	Amp               *uint256.Int
	SwapFeePercentage *uint256.Int
}

// https://etherscan.io/address/0x2ba7aa2213fa2c909cd9e46fed5a0059542b36b0#code#F10#L49
func (s *RegularSimulator) swap(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, *poolpkg.TokenAmount, *SwapInfo, error) {
	feeAmount, err := math.FixedPoint.MulUp(amountIn, s.SwapFeePercentage)
	if err != nil {
		return nil, nil, nil, err
	}
	amountInAfterFee, err := math.FixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, nil, nil, err
	}

	balances, err = _upscaleArray(balances, s.ScalingFactors)
	if err != nil {
		return nil, nil, nil, err
	}

	upScaledAmountInAfterFee, err := _upscale(amountInAfterFee, s.ScalingFactors[indexIn])
	if err != nil {
		return nil, nil, nil, err
	}

	upscaledAmountOut, err := s._onSwapGivenIn(upScaledAmountInAfterFee, balances, indexIn, indexOut)
	if err != nil {
		return nil, nil, nil, err
	}

	amountOut, err := _downscaleDown(upscaledAmountOut, s.ScalingFactors[indexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	fee := poolpkg.TokenAmount{
		Token:  s.Info.Tokens[indexIn],
		Amount: feeAmount.ToBig(),
	}

	return amountOut, &fee, &SwapInfo{}, nil
}

func (s *RegularSimulator) _onSwapGivenIn(
	amountIn *uint256.Int,
	balances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	return s._onRegularSwap(amountIn, balances, indexIn, indexOut)
}

func (s *RegularSimulator) _onRegularSwap(
	amountIn *uint256.Int,
	registeredBalances []*uint256.Int,
	indexIn int,
	indexOut int,
) (*uint256.Int, error) {
	balances := _dropBptItem(registeredBalances, s.BptIndex)
	indexIn, indexOut = _skipBptIndex(indexIn, s.BptIndex), _skipBptIndex(indexOut, s.BptIndex)

	invariant, err := math.StableMath.CalculateInvariantV2(s.Amp, balances)
	if err != nil {
		return nil, err
	}

	return math.StableMath.CalcOutGivenIn(
		invariant,
		s.Amp,
		amountIn,
		balances,
		indexIn,
		indexOut,
	)
}

func (s *RegularSimulator) updateBalance(params poolpkg.UpdateBalanceParams) {
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
