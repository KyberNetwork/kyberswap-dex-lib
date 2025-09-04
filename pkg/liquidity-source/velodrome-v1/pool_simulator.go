package velodromev1

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	isPaused     bool
	stable       bool
	fee          *uint256.Int
	feePrecision *uint256.Int
	reserves     []*uint256.Int
	decimals     []*uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	bigReserves := make([]*big.Int, len(entityPool.Reserves))
	reserves := make([]*uint256.Int, len(entityPool.Reserves))
	decimals := make([]*uint256.Int, len(entityPool.Tokens))
	for i, reserveStr := range entityPool.Reserves {
		token := entityPool.Tokens[i]
		tokens[i] = token.Address
		bigReserves[i] = bignumber.NewBig10(reserveStr)
		var err error
		if reserves[i], err = big256.NewUint256(reserveStr); err != nil {
			return nil, ErrInvalidReserve
		}
		decimals[i] = big256.TenPow(token.Decimals)
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    bigReserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		isPaused:     extra.IsPaused,
		stable:       staticExtra.Stable,
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(staticExtra.FeePrecision),
		reserves:     reserves,
		decimals:     decimals,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.isPaused {
		return nil, ErrPoolIsPaused
	}
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	var feeAmount uint256.Int
	feeAmount.Div(feeAmount.Mul(amountIn, p.fee), p.feePrecision)
	amountInAfterFee := amountIn.Sub(amountIn, &feeAmount)

	amountOut, err := p.getAmountOut(amountInAfterFee, indexIn, indexOut)
	if err != nil {
		return nil, err
	} else if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	} else if amountOut.Cmp(p.reserves[indexOut]) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            defaultGas + extraGasByExchange[p.GetExchange()],
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.isPaused {
		return nil, ErrPoolIsPaused
	}
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	} else if amountOut.Cmp(p.reserves[indexOut]) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn, err := p.getAmountIn(amountOut, indexIn, indexOut)
	if err != nil {
		return nil, err
	} else if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: params.TokenAmountOut.Token, Amount: bignumber.ZeroBI},
		Gas: defaultGas + extraGasByExchange[p.GetExchange()],
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserves = slices.Clone(p.reserves)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	amtIn, amtOut := uint256.MustFromBig(params.TokenAmountIn.Amount), uint256.MustFromBig(params.TokenAmountOut.Amount)
	p.reserves[indexIn] = amtIn.Add(p.reserves[indexIn], amtIn.Sub(amtIn, uint256.MustFromBig(params.Fee.Amount)))
	p.reserves[indexOut] = amtOut.Sub(p.reserves[indexOut], amtOut)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	exchange := p.GetExchange()
	return PoolMeta{
		Fee:          p.fee.Uint64(),
		FeePrecision: p.feePrecision.Uint64(),
		Stable:       p.stable,
		ApprovalInfo: pool.ApprovalInfo{
			ApprovalAddress: routerAddressByExchange[exchange],
		},
	}
}

func (p *PoolSimulator) getAmountOut(amountIn *uint256.Int, indexIn, indexOut int) (*uint256.Int, error) {
	if p.stable {
		xy := p._k(p.reserves[0], p.reserves[1])
		decimalsA, decimalsB := p.decimals[indexIn], p.decimals[indexOut]
		var _reserveIn, _reserveOut uint256.Int
		_reserveIn.Div(_reserveIn.Mul(p.reserves[indexIn], big256.BONE), decimalsA)
		_reserveOut.Div(_reserveOut.Mul(p.reserves[indexOut], big256.BONE), decimalsB)

		amountIn = new(uint256.Int).Mul(amountIn, big256.BONE)
		amountIn.Div(amountIn, decimalsA)
		y, err := p._get_y(_reserveIn.Add(amountIn, &_reserveIn), xy, &_reserveOut)
		if err != nil {
			return nil, err
		}
		y = y.Sub(&_reserveOut, y)

		return y.Div(y.Mul(y, decimalsB), big256.BONE), nil
	}

	var amountOut uint256.Int
	amountOut.MulDivOverflow(amountIn, p.reserves[indexOut], amountOut.Add(p.reserves[indexIn], amountIn))
	return &amountOut, nil
}

func (p *PoolSimulator) getAmountIn(amountOut *uint256.Int, indexIn, indexOut int) (*uint256.Int, error) {
	if p.stable {
		return nil, ErrUnimplemented
	}

	var tmp1, tmp2 uint256.Int
	tmp2.Mul(tmp1.Sub(p.reserves[indexOut], amountOut), tmp2.Sub(p.feePrecision, p.fee))
	tmp1.MulDivOverflow(tmp1.Mul(p.reserves[indexIn], amountOut), p.feePrecision, &tmp2)
	return tmp1.AddUint64(&tmp1, 1), nil
}

func (p *PoolSimulator) _k(x *uint256.Int, y *uint256.Int) *uint256.Int {
	if p.stable {
		var _x, _y, _a uint256.Int
		_x.Div(_x.Mul(x, big256.BONE), p.decimals[0])
		_y.Div(_y.Mul(y, big256.BONE), p.decimals[1])
		_a.Div(_a.Mul(&_x, &_y), big256.BONE)
		_b := _x.Add(
			_x.Div(
				_x.Mul(&_x, &_x),
				big256.BONE,
			),
			_y.Div(
				_y.Mul(&_y, &_y),
				big256.BONE,
			),
		)
		return _a.Div(_a.Mul(&_a, _b), big256.BONE)
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
				dy.Mul(dy.Sub(xy, k), big256.BONE),
				_d(x0, y),
			)
			y.Add(y, &dy)
		} else {
			dy.Div(
				dy.Mul(dy.Sub(k, xy), big256.BONE),
				_d(x0, y),
			)
			y.Sub(y, &dy)
		}

		if dy.CmpUint64(1) <= 0 {
			return y, nil
		}
	}

	return y, nil
}

func _f(x0 *uint256.Int, y *uint256.Int) *uint256.Int {
	var _a, _b uint256.Int
	_b.Add(
		_a.Div(
			_a.Mul(x0, x0),
			big256.BONE,
		),
		_b.Div(
			_b.Mul(y, y),
			big256.BONE,
		),
	)
	_a.Div(_a.Mul(x0, y), big256.BONE)
	return _a.Div(_a.Mul(&_a, &_b), big256.BONE)
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
				b.Div(b.Mul(y, y), big256.BONE),
			),
			big256.BONE,
		),
		b.Div(
			b.Mul(
				b.Div(b.Mul(x0, x0), big256.BONE),
				x0),
			big256.BONE),
	)
}
