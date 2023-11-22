package composable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	poolpkg.Pool

	regularSimulator *regularSimulator
	bptSimulator     *bptSimulator

	vaultAddress string
	poolID       string

	swapFeePercentage                   *uint256.Int
	scalingFactors                      []*uint256.Int
	bptIndex                            *uint256.Int
	amp                                 *uint256.Int
	bptTotalSupply                      *uint256.Int
	protocolFeePercentageCacheSwapType  *uint256.Int
	protocolFeePercentageCacheYieldType *uint256.Int

	lastJoinExit                     LastJoinExitData
	rateProviders                    []string
	tokensExemptFromYieldProtocolFee []bool
	tokenRateCaches                  []TokenRateCache

	mapTokenAddressToIndex map[string]int

	poolTypeVersion int
}

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	indexIn := s.getTokenIndex(tokenAmountIn.Token)
	indexOut := s.getTokenIndex(tokenOut)
	if indexIn == unknownInt || indexOut == unknownInt {
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
		fee       *poolpkg.TokenAmount
		err       error
	)
	if tokenAmountIn.Token == s.Info.Address || tokenOut == s.Info.Address {
		amountOut, fee, err = s.bptSimulator.swap(amountIn, balances, indexIn, indexOut)
	} else {
		amountOut, fee, err = s.regularSimulator.swap(amountIn, balances, indexIn, indexOut)
	}
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: fee,
		Gas: DefaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) getTokenIndex(token string) int {
	idx, ok := s.mapTokenAddressToIndex[token]
	if !ok {
		return unknownInt
	}
	return idx
}

// func (s *PoolSimulator) _swapWithBpt(
// 	amountIn *uint256.Int,
// 	registeredBalances []*uint256.Int,
// 	indexIn int,
// 	indexOut int,
// ) (*uint256.Int, *poolpkg.TokenAmount, error) {
// 	var err error

// 	registeredBalances, err = _upscaleArray(registeredBalances, s.scalingFactors)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	amountIn, err = _upscale(amountIn, s.scalingFactors[indexIn])
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return nil, nil, nil

// 	// preJoinExitSupply, balances, currentAmp, preJoinExitInvariant, err := s._beforeJoinExit(registeredBalances)
// }

// func (s *PoolSimulator) _beforeJoinExit(registeredBalances []*uint256.Int) (
// 	*uint256.Int,
// 	[]*uint256.Int,
// 	*uint256.Int,
// 	*uint256.Int,
// 	error,
// ) {
// 	preJoinExitSupply, balances, oldAmpPreJoinExitInvariant, err := s._payProtocolFeesBeforeJoinExit(registeredBalances)
// 	if err != nil {
// 		return nil, nil, nil, nil, err
// 	}
// }

// func (s *PoolSimulator) _payProtocolFeesBeforeJoinExit(
// 	registeredBalances []*uint256.Int,
// ) (*uint256.Int, []*uint256.Int, *uint256.Int, error) {
// 	virtualSupply, balances, err := s._dropBptItemFromBalances(registeredBalances)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	expectedProtocolOwnershipPercentage, currentInvariantWithLastJoinExitAmp, err := s._getProtocolPoolOwnershipPercentage(balances)
// }

// func (s *PoolSimulator) _getProtocolPoolOwnershipPercentage(balances []*uint256.Int) (*uint256.Int, *uint256.Int, error) {

// }

// func (s *PoolSimulator) bptForPoolOwnershipPercentage(
// 	supply *uint256.Int,
// 	basePercentage *uint256.Int,
// ) (*uint256.Int, error) {
// 	if s.poolTypeVersion == poolTypeVersion1 {
// 		return s.bptForPoolOwnershipPercentageV1(supply, basePercentage)
// 	}
// 	return s.bptForPoolOwnershipPercentageV2(supply, basePercentage)
// }

// func (s *PoolSimulator) bptForPoolOwnershipPercentageV1(
// 	supply *uint256.Int,
// 	basePercentage *uint256.Int,
// ) (*uint256.Int, error) {
// 	u, err := math.FixedPoint.MulDown(supply, basePercentage)
// 	if err != nil {
// 		return nil, err
// 	}
// 	v := math.FixedPoint.Complement(basePercentage)
// 	return math.FixedPoint.DivDown(u, v)
// }

// func (s *PoolSimulator) bptForPoolOwnershipPercentageV2(
// 	supply *uint256.Int,
// 	basePercentage *uint256.Int,
// ) (*uint256.Int, error) {
// 	u, err := math.Math.Mul(supply, basePercentage)
// 	if err != nil {
// 		return nil, err
// 	}
// 	v := math.FixedPoint.Complement(basePercentage)
// 	return math.Math.DivDown(u, v)
// }

// func (s *PoolSimulator) _getGrowthInvariants(balances []*uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
// 	totalGrowthInvariant, err := math.StableMath.CalculateInvariantV2(s.amp, balances)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	if totalGrowthInvariant.Cmp(s.lastJoinExit.LastJoinExitAmplification) <= 0 {
// 		return totalGrowthInvariant, totalGrowthInvariant, totalGrowthInvariant, nil
// 	}

// 	adjustedBalances, err := s._getAdjustedBalances(balances)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	swapFeeGrowthInvariant, err := math.StableMath.CalculateInvariantV2(
// 		s.lastJoinExit.LastJoinExitAmplification,
// 		adjustedBalances,
// 	)
// }

// func (s *PoolSimulator) _getAdjustedBalances(balances []*uint256.Int) ([]*uint256.Int, error) {
// 	if s.poolTypeVersion == poolTypeVersion5 {
// 		return s._getAdjustedBalancesV2(balances)
// 	}
// 	return s._getAdjustedBalancesV1(balances)
// }

func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}

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
