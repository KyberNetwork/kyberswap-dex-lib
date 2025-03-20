package velodromev2

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrPoolIsPaused             = errors.New("pool is paused")
	ErrInvalidAmountIn          = errors.New("invalid amountIn")
	ErrInvalidAmountOut         = errors.New("invalid amountOut")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrK                        = errors.New("K")
	ErrY                        = errors.New("!Y")
	ErrUnimplemented            = errors.New("unimplemented")
)

type (
	PoolSimulator struct {
		pool.Pool

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

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:    entityPool.Address,
			ReserveUsd: entityPool.ReserveUsd,
			Exchange:   entityPool.Exchange,
			Type:       entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return utils.NewBig(item) }),
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

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.isPaused {
		return nil, ErrPoolIsPaused
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	var feeAmount uint256.Int
	feeAmount.Div(feeAmount.Mul(amountIn, p.fee), p.feePrecision)
	amountInAfterFee := new(uint256.Int).Sub(amountIn, &feeAmount)

	amountOut, err := p.getAmountOut(
		amountInAfterFee,
		params.TokenAmountIn.Token,
	)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.isPaused {
		return nil, ErrPoolIsPaused
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	amountIn, err := p.getAmountIn(
		amountOut,
		params.TokenAmountOut.Token,
	)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: params.TokenAmountOut.Token, Amount: integer.Zero()},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	p.Pool.Info.Reserves[indexIn] = new(big.Int).Sub(new(big.Int).Add(p.Pool.Info.Reserves[indexIn],
		params.TokenAmountIn.Amount), params.Fee.Amount)
	p.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(p.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          p.fee.Uint64(),
		FeePrecision: p.feePrecision.Uint64(),
		BlockNumber:  p.Pool.Info.BlockNumber,
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})
	return &cloned
}

func (p *PoolSimulator) getAmountOut(
	amountIn *uint256.Int,
	tokenIn string,
) (*uint256.Int, error) {
	reserve0, overflow := uint256.FromBig(p.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserve1, overflow := uint256.FromBig(p.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountOut, err := p._getAmountOut(amountIn, tokenIn, reserve0, reserve1)
	if err != nil {
		return nil, err
	}

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	if tokenIn == p.Info.Tokens[0] && amountOut.Cmp(reserve1) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	if tokenIn == p.Info.Tokens[1] && amountOut.Cmp(reserve0) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	var balance0, balance1 *uint256.Int
	if tokenIn == p.Info.Tokens[0] {
		balance0 = new(uint256.Int).Add(reserve0, amountIn)
		balance1 = new(uint256.Int).Sub(reserve1, amountOut)
	} else {
		balance0 = new(uint256.Int).Sub(reserve0, amountOut)
		balance1 = new(uint256.Int).Add(reserve1, amountIn)
	}

	if p._k(balance0, balance1).Cmp(p._k(reserve0, reserve1)) < 0 {
		return nil, ErrK
	}

	return amountOut, nil
}

func (p *PoolSimulator) _getAmountOut(
	amountIn *uint256.Int,
	tokenIn string,
	_reserve0 *uint256.Int,
	_reserve1 *uint256.Int,
) (*uint256.Int, error) {
	if p.stable {
		xy := p._k(_reserve0, _reserve1)
		var _reserveA, _reserveB uint256.Int
		_reserveA.Div(_reserveA.Mul(_reserve0, number.Number_1e18), p.decimals0)
		_reserveB.Div(_reserveB.Mul(_reserve1, number.Number_1e18), p.decimals1)
		decimalsA, decimalsB := p.decimals0, p.decimals1

		if tokenIn != p.Info.Tokens[0] {
			_reserveA, _reserveB = _reserveB, _reserveA
			decimalsA, decimalsB = decimalsB, decimalsA
		}

		amountIn = new(uint256.Int).Mul(amountIn, number.Number_1e18)
		amountIn.Div(amountIn, decimalsA)
		y, err := p._get_y(_reserveA.Add(amountIn, &_reserveA), xy, &_reserveB)
		if err != nil {
			return nil, err
		}
		y = y.Sub(&_reserveB, y)

		return y.Div(y.Mul(y, decimalsB), number.Number_1e18), nil
	}

	var amountOut, newReserve uint256.Int
	if tokenIn == p.Info.Tokens[0] {
		return amountOut.Div(amountOut.Mul(amountIn, _reserve1), newReserve.Add(_reserve0, amountIn)), nil
	}
	return amountOut.Div(amountOut.Mul(amountIn, _reserve0), newReserve.Add(_reserve1, amountIn)), nil
}

func (p *PoolSimulator) getAmountIn(
	amountOut *uint256.Int,
	tokenOut string,
) (*uint256.Int, error) {
	reserve0, overflow := uint256.FromBig(p.Info.Reserves[0])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserve1, overflow := uint256.FromBig(p.Info.Reserves[1])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if tokenOut == p.Info.Tokens[0] && amountOut.Cmp(reserve0) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	if tokenOut == p.Info.Tokens[1] && amountOut.Cmp(reserve1) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn, err := p._getAmountIn(amountOut, tokenOut, reserve0, reserve1)
	if err != nil {
		return nil, err
	}

	if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	var balance0, balance1 *uint256.Int
	if tokenOut == p.Info.Tokens[0] {
		balance0 = new(uint256.Int).Sub(reserve0, amountOut)
		balance1 = new(uint256.Int).Add(reserve1, amountIn)
	} else {
		balance0 = new(uint256.Int).Add(reserve0, amountIn)
		balance1 = new(uint256.Int).Sub(reserve1, amountOut)
	}

	if p._k(balance0, balance1).Cmp(p._k(reserve0, reserve1)) < 0 {
		return nil, ErrK
	}

	return amountIn, nil
}

func (p *PoolSimulator) _getAmountIn(
	amountOut *uint256.Int,
	tokenOut string,
	_reserve0 *uint256.Int,
	_reserve1 *uint256.Int,
) (amountIn *uint256.Int, err error) {
	if p.stable {
		return nil, ErrUnimplemented
	}

	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	var reserveIn, reserveOut *uint256.Int
	if tokenOut == p.Info.Tokens[0] {
		reserveIn = _reserve1
		reserveOut = _reserve0
	} else {
		reserveIn = _reserve0
		reserveOut = _reserve1
	}

	numerator := SafeMul(
		SafeMul(reserveIn, amountOut),
		p.feePrecision,
	)
	denominator := SafeMul(
		SafeSub(reserveOut, amountOut),
		SafeSub(p.feePrecision, p.fee),
	)

	return SafeAdd(numerator.Div(numerator, denominator), number.Number_1), nil
}

func (p *PoolSimulator) _k(x *uint256.Int, y *uint256.Int) *uint256.Int {
	if p.stable {
		var _x, _y, _a uint256.Int
		_x.Div(_x.Mul(x, number.Number_1e18), p.decimals0)
		_y.Div(_y.Mul(y, number.Number_1e18), p.decimals1)
		_a.Div(_a.Mul(&_x, &_y), number.Number_1e18)
		_b := _x.Add(
			_x.Div(
				_x.Mul(&_x, &_x),
				number.Number_1e18,
			),
			_y.Div(
				_y.Mul(&_y, &_y),
				number.Number_1e18,
			),
		)
		return _a.Div(_a.Mul(&_a, _b), number.Number_1e18)
	}

	return new(uint256.Int).Mul(x, y)
}

func (p *PoolSimulator) _get_y(x0 *uint256.Int, xy *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	var dy uint256.Int
	y = y.Clone()
	for range 255 {
		k := _f(x0, y)

		if k.Cmp(xy) < 0 {
			dy.Div(
				dy.Mul(dy.Sub(xy, k), number.Number_1e18),
				_d(x0, y),
			)
			if dy.Sign() == 0 {
				if k.Cmp(xy) == 0 {
					return y, nil
				}
				if y := new(uint256.Int).AddUint64(y, 1); p._k(x0, y).Cmp(xy) > 0 {
					return y, nil
				}
				dy.SetOne()
			}
			y.Add(y, &dy)
		} else {
			dy.Div(
				dy.Mul(dy.Sub(k, xy), number.Number_1e18),
				_d(x0, y),
			)
			if dy.Sign() == 0 {
				if k.Cmp(xy) == 0 || _f(x0, new(uint256.Int).SubUint64(y, 1)).Cmp(xy) < 0 {
					return y, nil
				}
				dy.SetOne()
			}
			y.Sub(y, &dy)
		}
	}

	return nil, ErrY
}

func _f(x0 *uint256.Int, y *uint256.Int) *uint256.Int {
	var _a, _b uint256.Int
	_b.Add(
		_a.Div(
			_a.Mul(x0, x0),
			number.Number_1e18,
		),
		_b.Div(
			_b.Mul(y, y),
			number.Number_1e18,
		),
	)
	_a.Div(_a.Mul(x0, y), number.Number_1e18)
	return _a.Div(_a.Mul(&_a, &_b), number.Number_1e18)
}

func _d(x0 *uint256.Int, y *uint256.Int) *uint256.Int {
	var a, b uint256.Int
	return a.Add(
		a.Div(
			a.Mul(
				a.Mul(
					number.Number_3,
					x0,
				),
				b.Div(b.Mul(y, y), number.Number_1e18),
			),
			number.Number_1e18,
		),
		b.Div(
			b.Mul(
				b.Div(b.Mul(x0, x0), number.Number_1e18),
				x0),
			number.Number_1e18),
	)
}
