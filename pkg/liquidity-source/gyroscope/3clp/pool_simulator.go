package gyro3clp

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrPoolPaused         = errors.New("pool is paused")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrNotFoundThirdToken = errors.New("not found third token")
)

type PoolSimulator struct {
	poolpkg.Pool

	paused            bool
	scalingFactors    []*uint256.Int
	swapFeePercentage *uint256.Int
	root3Alpha        *uint256.Int
	poolTokenInfos    []*PoolTokenInfo

	vault  string
	poolID string

	poolType    string
	poolTypeVer int
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrInvalidToken
	}

	scalingFactorTokenIn, scalingFactorTokenOut := s.scalingFactors[indexIn], s.scalingFactors[indexOut]

	balanceTokenIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenIn, err := s._upscale(balanceTokenIn, scalingFactorTokenIn)
	if err != nil {
		return nil, ErrInvalidReserve
	}

	balanceTokenOut, err = s._upscale(balanceTokenOut, scalingFactorTokenOut)
	if err != nil {
		return nil, ErrInvalidReserve
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

	amountInAfterFee, err = s._upscale(amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	virtualOffset, err := s._calculateVirtualOffset(
		indexIn, indexOut, balanceTokenIn, balanceTokenOut,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err := Gyro3CLPMath._calcOutGivenIn(
		balanceTokenIn, balanceTokenOut, amountInAfterFee, virtualOffset,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err = s._downscaleDown(amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            defaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) _calculateVirtualOffset(indexIn, indexOut int, balanceTokenIn, balanceTokenOut *uint256.Int) (*uint256.Int, error) {
	balances := make([]*uint256.Int, 3)
	balances[0] = balanceTokenIn
	balances[1] = balanceTokenOut

	indexToken3, scalingFactor3, err := s._getThirdToken(indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	balances[2], err = s._getScaledTokenBalance(indexToken3, scalingFactor3)
	if err != nil {
		return nil, err
	}

	return s._calculateVirtualOffset_2(balances)
}

func (s *PoolSimulator) _calculateVirtualOffset_2(balances []*uint256.Int) (*uint256.Int, error) {
	invariant, err := Gyro3CLPMath._calculateInvariant(balances, s.root3Alpha)
	if err != nil {
		return nil, err
	}

	return math.GyroFixedPoint.MulDown(invariant, s.root3Alpha)
}

func (s *PoolSimulator) _getScaledTokenBalance(idx int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	balance := new(uint256.Int).Add(
		s.poolTokenInfos[idx].Cash,
		s.poolTokenInfos[idx].Managed,
	)

	balance, err := math.GyroFixedPoint.MulDown(balance, scalingFactor)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (s *PoolSimulator) _getThirdToken(indexIn, indexOut int) (int, *uint256.Int, error) {
	for idx, token := range s.Info.Tokens {
		if token != s.Info.Tokens[indexIn] && token != s.Info.Tokens[indexOut] {
			return idx, s.scalingFactors[idx], nil
		}
	}
	return -1, nil, ErrNotFoundThirdToken
}

func (s *PoolSimulator) _upscale(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.MulDown(amount, scalingFactor)
}

func (s *PoolSimulator) _downscaleDown(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.DivDown(amount, scalingFactor)
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:       s.vault,
		PoolID:      s.poolID,
		T:           s.poolType,
		V:           s.poolTypeVer,
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
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
