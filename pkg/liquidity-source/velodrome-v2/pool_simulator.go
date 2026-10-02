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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
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

	var feeAmount, amountInAfterFee uint256.Int
	if err := checkedMulDiv(&feeAmount, amountIn, p.fee, p.feePrecision); err != nil {
		return nil, err
	}
	if err := checkedSub(&amountInAfterFee, amountIn, &feeAmount); err != nil {
		return nil, err
	}

	amountOut, err := p.getAmountOut(
		&amountInAfterFee,
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
	p.Info.Reserves[indexIn] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[indexIn],
		params.TokenAmountIn.Amount), params.Fee.Amount)
	p.Info.Reserves[indexOut] = new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		Fee:          p.fee.Uint64(),
		FeePrecision: p.feePrecision.Uint64(),
		BlockNumber:  p.Info.BlockNumber,
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

	if tokenIn == p.Info.Tokens[0] && !amountOut.Lt(reserve1) {
		return nil, ErrInsufficientLiquidity
	}

	if tokenIn == p.Info.Tokens[1] && !amountOut.Lt(reserve0) {
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

	kAfter, err := p._k(balance0, balance1)
	if err != nil {
		return nil, err
	}
	kBefore, err := p._k(reserve0, reserve1)
	if err != nil {
		return nil, err
	}
	if kAfter.Cmp(kBefore) < 0 {
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
		xy, err := p._k(_reserve0, _reserve1)
		if err != nil {
			return nil, err
		}
		var _reserveA, _reserveB uint256.Int
		if err = checkedMulDiv(&_reserveA, _reserve0, number.Number_1e18, p.decimals0); err != nil {
			return nil, err
		}
		if err = checkedMulDiv(&_reserveB, _reserve1, number.Number_1e18, p.decimals1); err != nil {
			return nil, err
		}
		decimalsA, decimalsB := p.decimals0, p.decimals1

		if tokenIn != p.Info.Tokens[0] {
			_reserveA, _reserveB = _reserveB, _reserveA
			decimalsA, decimalsB = decimalsB, decimalsA
		}

		var scaledAmountIn, x0 uint256.Int
		if err = checkedMulDiv(&scaledAmountIn, amountIn, number.Number_1e18, decimalsA); err != nil {
			return nil, err
		}
		if err = checkedAdd(&x0, &scaledAmountIn, &_reserveA); err != nil {
			return nil, err
		}
		y, err := p._get_y(&x0, xy, &_reserveB)
		if err != nil {
			return nil, err
		}
		if err = checkedSub(y, &_reserveB, y); err != nil {
			return nil, err
		}
		if err = checkedMulDiv(y, y, decimalsB, number.Number_1e18); err != nil {
			return nil, err
		}

		return y, nil
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

	if tokenOut == p.Info.Tokens[0] && !amountOut.Lt(reserve0) {
		return nil, ErrInsufficientLiquidity
	}

	if tokenOut == p.Info.Tokens[1] && !amountOut.Lt(reserve1) {
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

	// Skip K invariant check for stable pools since the invariant is different
	if !p.stable {
		kAfter, err := p._k(balance0, balance1)
		if err != nil {
			return nil, err
		}
		kBefore, err := p._k(reserve0, reserve1)
		if err != nil {
			return nil, err
		}
		if kAfter.Cmp(kBefore) < 0 {
			return nil, ErrK
		}
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
		xy, err := p._k(_reserve0, _reserve1)
		if err != nil {
			return nil, err
		}
		var tmp uint256.Int
		_reserveA := big256.MulDivDown(new(uint256.Int), _reserve0, number.Number_1e18, p.decimals0)
		_reserveB := big256.MulDivDown(new(uint256.Int), _reserve1, number.Number_1e18, p.decimals1)

		if tokenOut == p.Info.Tokens[0] {
			amountOutScaled := big256.MulDivUp(&tmp, amountOut, number.Number_1e18, p.decimals0)
			newReserveA := new(uint256.Int)
			if err = checkedSub(newReserveA, _reserveA, amountOutScaled); err != nil {
				return nil, err
			}
			x, err := p._get_y(newReserveA, xy, _reserveB)
			if err != nil {
				return nil, err
			}
			amountIn = new(uint256.Int)
			if err = checkedSub(amountIn, x, _reserveB); err != nil {
				return nil, err
			}
			tmp.Sub(p.feePrecision, p.fee)
			amountIn = big256.MulDivUp(&tmp, amountIn, p.feePrecision, &tmp)
			return big256.MulWadUp(&tmp, amountIn, p.decimals1), nil
		}

		amountOutScaled := big256.MulDivUp(&tmp, amountOut, number.Number_1e18, p.decimals1)
		newReserveB := new(uint256.Int)
		if err = checkedSub(newReserveB, _reserveB, amountOutScaled); err != nil {
			return nil, err
		}
		x, err := p._get_y(newReserveB, xy, _reserveA)
		if err != nil {
			return nil, err
		}
		amountIn = new(uint256.Int)
		if err = checkedSub(amountIn, x, _reserveA); err != nil {
			return nil, err
		}
		tmp.Sub(p.feePrecision, p.fee)
		amountIn = big256.MulDivUp(&tmp, amountIn, p.feePrecision, &tmp)
		return big256.MulWadUp(&tmp, amountIn, p.decimals0), nil
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

func (p *PoolSimulator) _k(x *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	if !p.stable {
		return new(uint256.Int).Mul(x, y), nil
	}

	var _x, _y, _a, _b, t uint256.Int
	if err := checkedMulDiv(&_x, x, number.Number_1e18, p.decimals0); err != nil {
		return nil, err
	}
	if err := checkedMulDiv(&_y, y, number.Number_1e18, p.decimals1); err != nil {
		return nil, err
	}
	if err := checkedMulDiv(&_a, &_x, &_y, number.Number_1e18); err != nil {
		return nil, err
	}
	if err := checkedMulDiv(&_b, &_x, &_x, number.Number_1e18); err != nil {
		return nil, err
	}
	if err := checkedMulDiv(&t, &_y, &_y, number.Number_1e18); err != nil {
		return nil, err
	}
	if err := checkedAdd(&_b, &_b, &t); err != nil {
		return nil, err
	}
	if err := checkedMulDiv(&_a, &_a, &_b, number.Number_1e18); err != nil {
		return nil, err
	}

	return &_a, nil
}

func (p *PoolSimulator) _get_y(x0 *uint256.Int, xy *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	var dy, yNext uint256.Int
	y = y.Clone()
	for range 255 {
		k, err := _f(x0, y)
		if err != nil {
			return nil, err
		}

		if k.Lt(xy) {
			dy.Sub(xy, &k)
			if err = checkedMul(&dy, &dy, number.Number_1e18); err != nil {
				return nil, err
			}
			d, err := _d(x0, y)
			if err != nil {
				return nil, err
			}
			if err = checkedDiv(&dy, &dy, &d); err != nil {
				return nil, err
			}
			if dy.IsZero() {
				if k.Eq(xy) {
					return y, nil
				}
				if err = checkedAdd(&yNext, y, number.Number_1); err != nil {
					return nil, err
				}
				kNext, err := p._k(x0, &yNext)
				if err != nil {
					return nil, err
				}
				if kNext.Gt(xy) {
					return &yNext, nil
				}
				dy.SetOne()
			}
			if err = checkedAdd(y, y, &dy); err != nil {
				return nil, err
			}
		} else {
			dy.Sub(&k, xy)
			if err = checkedMul(&dy, &dy, number.Number_1e18); err != nil {
				return nil, err
			}
			d, err := _d(x0, y)
			if err != nil {
				return nil, err
			}
			if err = checkedDiv(&dy, &dy, &d); err != nil {
				return nil, err
			}
			if dy.IsZero() {
				if k.Eq(xy) {
					return y, nil
				}
				if err = checkedSub(&yNext, y, number.Number_1); err != nil {
					return nil, err
				}
				kPrev, err := _f(x0, &yNext)
				if err != nil {
					return nil, err
				}
				if kPrev.Lt(xy) {
					return y, nil
				}
				dy.SetOne()
			}
			if err = checkedSub(y, y, &dy); err != nil {
				return nil, err
			}
		}
	}

	return nil, ErrY
}

func _f(x0 *uint256.Int, y *uint256.Int) (res uint256.Int, err error) {
	var _a, _b, t uint256.Int
	if err = checkedMulDiv(&_a, x0, y, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&_b, x0, x0, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&t, y, y, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedAdd(&_b, &_b, &t); err != nil {
		return res, err
	}
	err = checkedMulDiv(&res, &_a, &_b, number.Number_1e18)
	return res, err
}

func _d(x0 *uint256.Int, y *uint256.Int) (res uint256.Int, err error) {
	var a, b uint256.Int
	if err = checkedMul(&a, number.Number_3, x0); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&b, y, y, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&a, &a, &b, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&b, x0, x0, number.Number_1e18); err != nil {
		return res, err
	}
	if err = checkedMulDiv(&b, &b, x0, number.Number_1e18); err != nil {
		return res, err
	}
	err = checkedAdd(&res, &a, &b)
	return res, err
}
