package uniswapv2

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                = errors.New("K")
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		fee          *uint256.Int
		feePrecision *uint256.Int

		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

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
		fee:          uint256.NewInt(staticExtra.Fee),
		feePrecision: uint256.NewInt(staticExtra.FeePrecision),
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(tokenAmountIn poolpkg.TokenAmount, tokenOut string) (*poolpkg.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(tokenAmountIn.Token)
	indexOut := s.GetTokenIndex(tokenOut)

	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.Cmp(zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn := uint256.MustFromBig(s.Pool.Info.Reserves[indexIn])
	reserveOut := uint256.MustFromBig(s.Pool.Info.Reserves[indexOut])

	if reserveIn.Cmp(zero) <= 0 || reserveOut.Cmp(zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)

	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	balance0 := new(uint256.Int).Add(reserveIn, amountIn)
	balance1 := new(uint256.Int).Sub(reserveOut, amountOut)

	balance0Adjusted := new(uint256.Int).Sub(
		new(uint256.Int).Mul(balance0, s.feePrecision),
		new(uint256.Int).Mul(amountIn, s.fee),
	)
	balance1Adjusted := new(uint256.Int).Mul(balance1, s.feePrecision)

	kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(uint256.Int).Mul(balance0Adjusted, balance1Adjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it zero to avoid null pointer exception
		Fee: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          s.fee.Uint64(),
		FeePrecision: s.feePrecision.Uint64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Mul(amountIn, new(uint256.Int).Sub(s.feePrecision, s.fee))
	numerator := new(uint256.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, s.feePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator)
}
