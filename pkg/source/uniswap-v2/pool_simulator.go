package uniswapv2

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/samber/lo"

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
		fee          *big.Int
		feePrecision *big.Int

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
		fee:          big.NewInt(staticExtra.Fee),
		feePrecision: big.NewInt(staticExtra.FeePrecision),
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(tokenAmountIn poolpkg.TokenAmount, tokenOut string) (*poolpkg.CalcAmountOutResult, error) {
	if tokenAmountIn.Token == s.Pool.Info.Tokens[0] && tokenOut == s.Pool.Info.Tokens[1] {
		return s.swap0to1(tokenAmountIn.Amount)
	}

	if tokenAmountIn.Token == s.Pool.Info.Tokens[1] && tokenOut == s.Pool.Info.Tokens[0] {
		return s.swap1to0(tokenAmountIn.Amount)
	}

	return nil, ErrInvalidToken
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	if params.TokenAmountIn.Token == s.Pool.Info.Tokens[0] && params.TokenAmountOut.Token == s.Pool.Info.Tokens[1] {
		s.Pool.Info.Reserves[0] = new(big.Int).Add(s.Pool.Info.Reserves[0], params.TokenAmountIn.Amount)
		s.Pool.Info.Reserves[1] = new(big.Int).Sub(s.Pool.Info.Reserves[1], params.TokenAmountOut.Amount)

		return
	}

	if params.TokenAmountIn.Token == s.Pool.Info.Tokens[1] && params.TokenAmountOut.Token == s.Pool.Info.Tokens[0] {
		s.Pool.Info.Reserves[0] = new(big.Int).Sub(s.Pool.Info.Reserves[0], params.TokenAmountOut.Amount)
		s.Pool.Info.Reserves[1] = new(big.Int).Add(s.Pool.Info.Reserves[1], params.TokenAmountIn.Amount)

		return
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          s.fee.Int64(),
		FeePrecision: s.feePrecision.Int64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) swap0to1(amountIn *big.Int) (*poolpkg.CalcAmountOutResult, error) {
	if amountIn.Cmp(integer.Zero()) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, reserveOut := s.Pool.Info.Reserves[0], s.Pool.Info.Reserves[1]

	if reserveIn.Cmp(integer.Zero()) <= 0 || reserveOut.Cmp(integer.Zero()) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)

	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	balance0 := new(big.Int).Add(reserveIn, amountIn)
	balance1 := new(big.Int).Sub(reserveOut, amountOut)

	balance0Adjusted := new(big.Int).Sub(
		new(big.Int).Mul(balance0, s.feePrecision),
		new(big.Int).Mul(amountIn, s.fee),
	)
	balance1Adjusted := new(big.Int).Mul(balance1, s.feePrecision)

	kBefore := new(big.Int).Mul(new(big.Int).Mul(reserveIn, reserveOut), new(big.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(big.Int).Mul(balance0Adjusted, balance1Adjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[1], Amount: amountOut},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) swap1to0(amountIn *big.Int) (*poolpkg.CalcAmountOutResult, error) {
	if amountIn.Cmp(integer.Zero()) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, reserveOut := s.Pool.Info.Reserves[1], s.Pool.Info.Reserves[0]

	if reserveIn.Cmp(integer.Zero()) <= 0 || reserveOut.Cmp(integer.Zero()) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)

	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	balance0 := new(big.Int).Sub(reserveOut, amountOut)
	balance1 := new(big.Int).Add(reserveIn, amountIn)

	balance0Adjusted := new(big.Int).Mul(balance0, s.feePrecision)
	balance1Adjusted := new(big.Int).Sub(
		new(big.Int).Mul(balance1, s.feePrecision),
		new(big.Int).Mul(amountIn, s.fee),
	)

	kBefore := new(big.Int).Mul(new(big.Int).Mul(reserveIn, reserveOut), new(big.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(big.Int).Mul(balance0Adjusted, balance1Adjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[0], Amount: amountOut},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) getAmountOut(amountIn *big.Int, reserveIn *big.Int, reserveOut *big.Int) *big.Int {
	amountInWithFee := new(big.Int).Mul(amountIn, new(big.Int).Sub(s.feePrecision, s.fee))
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, s.feePrecision), amountInWithFee)

	return new(big.Int).Div(numerator, denominator)
}
