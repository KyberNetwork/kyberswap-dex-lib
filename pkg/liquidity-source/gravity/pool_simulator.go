package gravity

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/blockchain-toolkit/number"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                = errors.New("K")
)

var (
	getAmountOutRemainAfterFee = uint256.NewInt(997)
	getAmountOutFeePrecision   = uint256.NewInt(1000)
	fee                        = uint256.NewInt(25)
	feePrecision               = uint256.NewInt(10000)
	remainAfterGovFee          = uint256.NewInt(9995)
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOutBeforeGovFee := s.getAmountOut(amountIn, reserveIn, reserveOut)
	amountOut := new(uint256.Int).Div(new(uint256.Int).Mul(amountOutBeforeGovFee, remainAfterGovFee), feePrecision)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	balanceInAdjusted := new(uint256.Int).Sub(
		new(uint256.Int).Mul(balanceIn, feePrecision),
		new(uint256.Int).Mul(amountIn, fee),
	)
	balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, feePrecision)

	kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(feePrecision, feePrecision))
	kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: new(uint256.Int).Sub(amountOutBeforeGovFee, amountOut).ToBig()},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount), params.Fee.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          3,
		FeePrecision: 10000,
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Mul(amountIn, getAmountOutRemainAfterFee)
	numerator := new(uint256.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, getAmountOutFeePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator)
}
