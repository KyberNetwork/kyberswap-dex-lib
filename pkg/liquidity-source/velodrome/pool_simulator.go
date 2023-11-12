package velodrome

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPoolIsPaused             = errors.New("pool is paused")
	ErrInvalidAmountIn          = errors.New("invalid amountIn")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrK                        = errors.New("K")
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		stable       bool
		decimals0    *uint256.Int
		decimals1    *uint256.Int
		feePrecision *uint256.Int

		isPaused bool
		fee      *uint256.Int

		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
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

		stable:       staticExtra.Stable,
		decimals0:    staticExtra.Decimal0,
		decimals1:    staticExtra.Decimal1,
		feePrecision: uint256.NewInt(staticExtra.FeePrecision),

		isPaused: extra.IsPaused,
		fee:      uint256.NewInt(extra.Fee),

		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	if s.isPaused {
		return nil, ErrPoolIsPaused
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	feeAmount := new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, s.fee), s.feePrecision)
	amountInAfterFee := new(uint256.Int).Sub(amountIn, feeAmount)

	amountOut, err := s.getAmountOut(
		amountInAfterFee,
		tokenAmountIn.Token,
	)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: tokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	s.Pool.Info.Reserves[indexIn] = new(big.Int).Sub(new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount), params.Fee.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          s.fee.Uint64(),
		FeePrecision: s.feePrecision.Uint64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(
	amountIn *uint256.Int,
	tokenIn string,
) (*uint256.Int, error) {
	reserve0, overflow := uint256.FromBig(s.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserve1, overflow := uint256.FromBig(s.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountOut := s._getAmountOut(amountIn, tokenIn, reserve0, reserve1)

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	if tokenIn == s.Info.Tokens[0] && amountOut.Cmp(reserve1) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	if tokenIn == s.Info.Tokens[1] && amountOut.Cmp(reserve0) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	var balance0, balance1 *uint256.Int
	if tokenIn == s.Info.Tokens[0] {
		balance0 = new(uint256.Int).Add(reserve0, amountIn)
		balance1 = new(uint256.Int).Sub(reserve1, amountOut)
	} else {
		balance0 = new(uint256.Int).Sub(reserve0, amountOut)
		balance1 = new(uint256.Int).Add(reserve1, amountIn)
	}

	if s._k(balance0, balance1).Cmp(s._k(reserve0, reserve1)) < 0 {
		return nil, ErrK
	}

	return amountOut, nil
}

func (s *PoolSimulator) _getAmountOut(
	amountIn *uint256.Int,
	tokenIn string,
	_reserve0 *uint256.Int,
	_reserve1 *uint256.Int,
) *uint256.Int {
	if s.stable {
		xy := s._k(_reserve0, _reserve1)
		_reserve0 = new(uint256.Int).Div(new(uint256.Int).Mul(_reserve0, number.Number_1e18), s.decimals0)
		_reserve1 = new(uint256.Int).Div(new(uint256.Int).Mul(_reserve1, number.Number_1e18), s.decimals1)

		if tokenIn == s.Info.Tokens[0] {
			amountIn = new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, number.Number_1e18), s.decimals0)
			y := new(uint256.Int).Sub(_reserve1, s._get_y(new(uint256.Int).Add(amountIn, _reserve0), xy, _reserve1))
			return new(uint256.Int).Div(new(uint256.Int).Mul(y, s.decimals1), number.Number_1e18)
		}

		amountIn = new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, number.Number_1e18), s.decimals1)
		y := new(uint256.Int).Sub(_reserve0, s._get_y(new(uint256.Int).Add(amountIn, _reserve1), xy, _reserve0))
		return new(uint256.Int).Div(new(uint256.Int).Mul(y, s.decimals0), number.Number_1e18)
	}

	if tokenIn == s.Info.Tokens[0] {
		return new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, _reserve1), new(uint256.Int).Add(_reserve0, amountIn))
	}

	return new(uint256.Int).Div(new(uint256.Int).Mul(amountIn, _reserve0), new(uint256.Int).Add(_reserve1, amountIn))
}

func (s *PoolSimulator) _k(x *uint256.Int, y *uint256.Int) *uint256.Int {
	if s.stable {
		_x := new(uint256.Int).Div(new(uint256.Int).Mul(x, number.Number_1e18), s.decimals0)
		_y := new(uint256.Int).Div(new(uint256.Int).Mul(y, number.Number_1e18), s.decimals1)
		_a := new(uint256.Int).Div(new(uint256.Int).Mul(_x, _y), number.Number_1e18)
		_b := new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(_x, _x),
				number.Number_1e18,
			),
			new(uint256.Int).Div(
				new(uint256.Int).Mul(_y, _y),
				number.Number_1e18,
			),
		)
		return new(uint256.Int).Div(new(uint256.Int).Mul(_a, _b), number.Number_1e18)
	}

	return new(uint256.Int).Mul(x, y)
}

func (s *PoolSimulator) _get_y(x0 *uint256.Int, xy *uint256.Int, y *uint256.Int) *uint256.Int {
	for i := 0; i < 255; i++ {
		y_prev := new(uint256.Int).Set(y)
		k := _f(x0, y)

		if k.Cmp(xy) < 0 {
			dy := new(uint256.Int).Div(
				new(uint256.Int).Mul(new(uint256.Int).Sub(xy, k), number.Number_1e18),
				_d(x0, y),
			)
			y = new(uint256.Int).Add(y, dy)
		} else {
			dy := new(uint256.Int).Div(
				new(uint256.Int).Mul(new(uint256.Int).Sub(k, xy), number.Number_1e18),
				_d(x0, y),
			)
			y = new(uint256.Int).Sub(y, dy)
		}

		if y.Cmp(y_prev) > 0 {
			if new(uint256.Int).Sub(y, y_prev).Cmp(number.Number_1) <= 0 {
				return y
			}
		} else {
			if new(uint256.Int).Sub(y_prev, y).Cmp(number.Number_1) <= 0 {
				return y
			}
		}
	}

	return y
}

func _f(x0 *uint256.Int, y *uint256.Int) *uint256.Int {
	_a := new(uint256.Int).Div(new(uint256.Int).Mul(x0, y), number.Number_1e18)
	_b := new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(x0, x0),
			number.Number_1e18,
		),
		new(uint256.Int).Div(
			new(uint256.Int).Mul(y, y),
			number.Number_1e18,
		),
	)
	return new(uint256.Int).Div(new(uint256.Int).Mul(_a, _b), number.Number_1e18)
}

func _d(x0 *uint256.Int, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				number.Number_3,
				new(uint256.Int).Mul(
					x0,
					new(uint256.Int).Div(new(uint256.Int).Mul(y, y), number.Number_1e18),
				),
			),
			number.Number_1e18,
		),
		new(uint256.Int).Mul(
			new(uint256.Int).Div(new(uint256.Int).Mul(x0, x0), number.Number_1e18),
			new(uint256.Int).Div(x0, number.Number_1e18),
		),
	)
}
