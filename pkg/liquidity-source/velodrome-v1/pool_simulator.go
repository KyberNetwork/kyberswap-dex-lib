package velodromev1

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	gas          int64

	// balances is balanceOf(pair) of each token (nil when unknown), see PoolExtra.Balance0.
	balances []*uint256.Int
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

	var balances []*uint256.Int
	if extra.Balance0 != nil && extra.Balance1 != nil && len(reserves) == 2 {
		balance0, overflow0 := uint256.FromBig(extra.Balance0)
		balance1, overflow1 := uint256.FromBig(extra.Balance1)
		if !overflow0 && !overflow1 { // a balance that does not fit 256 bits is treated as unknown
			balances = []*uint256.Int{balance0, balance1}
		}
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    bigReserves,
			BlockNumber: entityPool.BlockNumber,
		},
		isPaused:     extra.IsPaused,
		stable:       staticExtra.Stable,
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(staticExtra.FeePrecision),
		reserves:     reserves,
		decimals:     decimals,
		gas:          lo.CoalesceOrEmpty(staticExtra.Gas, defaultGas+extraGasByExchange[entityPool.Exchange]),
		balances:     balances,
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

	// The executor quotes from the amount the pair credits (balanceIn + sent - reserveIn), not from the amount sent;
	// so does this when the balances are known. The reported fee is taken on that credited amount, as getAmountOut does.
	credited := amountIn
	if p.creditsBalanceDelta() {
		var creditedAmount uint256.Int
		if err := p.credit(&creditedAmount, indexIn, amountIn); err != nil {
			return nil, err
		}
		credited = &creditedAmount
	}
	var feeAmount uint256.Int
	feeAmount.MulDivOverflow(credited, p.fee, p.feePrecision)
	var amountInAfterFee uint256.Int
	amountInAfterFee.Sub(credited, &feeAmount)

	amountOut, err := p.getAmountOut(&amountInAfterFee, indexIn, indexOut)
	if err != nil {
		return nil, err
	} else if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	} else if !amountOut.Lt(p.reserves[indexOut]) {
		return nil, ErrInsufficientLiquidity
	}

	swapInfo, err := p.checkSwap(indexIn, indexOut, amountIn, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            p.gas,
		SwapInfo:       swapInfo,
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
	} else if !amountOut.Lt(p.reserves[indexOut]) {
		return nil, ErrInsufficientLiquidity
	}

	// getAmountIn finds the amount the pair has to credit; the router sends that amount less what the pair already
	// credits (balanceIn - reserveIn), when the balances are known
	amountIn, err := p.getAmountIn(amountOut, indexIn, indexOut)
	if err != nil {
		return nil, err
	} else if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}
	if p.creditsBalanceDelta() {
		if err = p.sendFor(amountIn, indexIn); err != nil {
			return nil, err
		}
	}

	swapInfo, err := p.checkSwap(indexIn, indexOut, amountIn, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee:      &pool.TokenAmount{Token: params.TokenAmountOut.Token, Amount: bignumber.ZeroBI},
		Gas:      p.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserves = slices.Clone(p.reserves)
	cloned.balances = slices.Clone(p.balances)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}
	amtIn, amtOut := uint256.MustFromBig(params.TokenAmountIn.Amount), uint256.MustFromBig(params.TokenAmountOut.Amount)
	if p.balances != nil {
		// the pair syncs its reserves to the balances at the end of swap
		if info, ok := params.SwapInfo.(SwapInfo); ok {
			p.setBalances(&info.NewBalance0, &info.NewBalance1)
			return
		}
		// no usable SwapInfo (a result that did not come from this pool's CalcAmountOut/CalcAmountIn): swap again
		if post, err := p.swap(indexIn, indexOut, amtIn, amtOut); err == nil {
			p.setBalances(&post[0], &post[1])
			return
		}
		p.balances = nil // not a swap the pair accepts: stop trusting the balances and update as without them
	}
	p.reserves[indexIn] = amtIn.Add(p.reserves[indexIn], amtIn.Sub(amtIn, uint256.MustFromBig(params.Fee.Amount)))
	p.reserves[indexOut] = amtOut.Sub(p.reserves[indexOut], amtOut)
}

// setBalances stores the balances after a swap as both the balances and the reserves (the pair syncs them), as copies
// so that the caller's values are not shared with the state.
func (p *PoolSimulator) setBalances(balance0, balance1 *uint256.Int) {
	p.reserves = []*uint256.Int{balance0.Clone(), balance1.Clone()}
	p.balances = []*uint256.Int{balance0.Clone(), balance1.Clone()}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	exchange := p.GetExchange()
	return PoolMeta{
		Fee:             p.fee.Uint64(),
		FeePrecision:    p.feePrecision.Uint64(),
		Stable:          p.stable,
		ApprovalAddress: routerAddressByExchange[exchange],
		BlockNumber:     p.Info.BlockNumber,
	}
}

// creditsBalanceDelta tells whether the executor credits the pair with balanceIn + sent - reserveIn: it does for every
// exchange it swaps with directly (executeVelodrome, executeSolidlyV2, executeMetadex: amountIn =
// balanceOf(pair) - reserveIn after the transfer). The exchanges that swap through their own router (the ones with a
// router address) quote from the amount sent. Without the balances nothing is known to credit.
func (p *PoolSimulator) creditsBalanceDelta() bool {
	if p.balances == nil {
		return false
	}
	_, viaRouter := routerAddressByExchange[p.Info.Exchange]
	return !viaRouter
}

// credit sets credited to what the executor's pair is credited when the router sends sent:
// balanceIn + sent - reserveIn. The executor subtracts without a check, so a result below zero would wrap and
// the swap would revert; a zero is rejected by the pair. The caller checks that the balances are known.
func (p *PoolSimulator) credit(credited *uint256.Int, indexIn int, sent *uint256.Int) error {
	if _, overflow := credited.AddOverflow(p.balances[indexIn], sent); overflow {
		return ErrInvalidAmountIn
	} else if _, underflow := credited.SubOverflow(credited, p.reserves[indexIn]); underflow || credited.IsZero() {
		return ErrInsufficientInputAmount
	}
	return nil
}

// sendFor turns the amount to be credited, in place, into the amount the router has to send: the credited amount
// less the surplus of the pair on the input side (balanceIn - reserveIn), or plus its deficit. It is the inverse of
// credit. The caller checks that the balances are known.
func (p *PoolSimulator) sendFor(amount *uint256.Int, indexIn int) error {
	balanceIn, reserveIn := p.balances[indexIn], p.reserves[indexIn]
	if balanceIn.Lt(reserveIn) {
		var deficit uint256.Int
		if _, overflow := amount.AddOverflow(amount, deficit.Sub(reserveIn, balanceIn)); overflow {
			return ErrInvalidAmountIn
		}
		return nil
	}
	var surplus uint256.Int
	surplus.Sub(balanceIn, reserveIn)
	if !amount.Gt(&surplus) { // the pair already credits that much: nothing is left to send
		return ErrInsufficientInputAmount
	}
	amount.Sub(amount, &surplus)
	return nil
}

// checkSwap runs swap when the balances are known and returns the SwapInfo UpdateBalance needs, or a nil interface
// when the balances are unknown.
func (p *PoolSimulator) checkSwap(indexIn, indexOut int, amountIn, amountOut *uint256.Int) (any, error) {
	if p.balances == nil {
		return nil, nil
	}
	post, err := p.swap(indexIn, indexOut, amountIn, amountOut)
	if err != nil {
		return nil, err
	}
	return SwapInfo{NewBalance0: post[0], NewBalance1: post[1]}, nil
}

// swap mirrors the checks of DystPair.swap after the quote: the pair pays amountOut out of its balance, then takes
// as input whatever its balances hold above the reserves (this is how a donation, or a deficit, changes what the
// swapper is credited), moves the fee of the input out of the pair, and requires the curve not to decrease. It
// returns the balances after the swap, which the pair stores as its new reserves. amountIn is the amount the router
// sent to the pair (what the pair credits for it is derived here, from the balances); amountOut was quoted from the
// reserves and the credited amount. The caller checks that the balances are known.
// The results are values on the stack: this runs on every quote.
func (p *PoolSimulator) swap(indexIn, indexOut int, amountIn, amountOut *uint256.Int) (post [2]uint256.Int, err error) {
	reserveOut := p.reserves[indexOut]
	// require(amount0Out < _reserve0 && amount1Out < _reserve1)
	if !amountOut.Lt(reserveOut) {
		return post, ErrInsufficientLiquidity
	}
	// safeTransfer(to, amountOut)
	if p.balances[indexOut].Lt(amountOut) {
		return post, ErrInsufficientBalance
	}

	// the router has sent amountIn before the swap
	if _, overflow := post[indexIn].AddOverflow(p.balances[indexIn], amountIn); overflow {
		return post, ErrInvalidAmountIn
	}
	post[indexOut].Sub(p.balances[indexOut], amountOut)

	// amountXIn = balanceX > reserveX - amountXOut ? balanceX - (reserveX - amountXOut) : 0
	var reservesAfterOut, credited [2]uint256.Int
	reservesAfterOut[0].Set(p.reserves[0])
	reservesAfterOut[1].Set(p.reserves[1])
	reservesAfterOut[indexOut].Sub(reserveOut, amountOut)
	for i := range post {
		if post[i].Gt(&reservesAfterOut[i]) {
			credited[i].Sub(&post[i], &reservesAfterOut[i])
		}
	}
	// require(amount0In > 0 || amount1In > 0)
	if credited[0].IsZero() && credited[1].IsZero() {
		return post, ErrInsufficientInputAmount
	}

	// _update0/_update1 move the fee of the input out of the pair (to the treasury and the fees contract)
	for i := range post {
		var fee uint256.Int
		fee.MulDivOverflow(&credited[i], p.fee, p.feePrecision)
		post[i].Sub(&post[i], &fee)
	}

	// require(_k(balance0, balance1) >= _k(reserve0, reserve1))
	if p._k(&post[0], &post[1]).Lt(p._k(p.reserves[0], p.reserves[1])) {
		return post, ErrK
	}
	return post, nil
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
		xy := p._k(p.reserves[0], p.reserves[1])
		decimalsA, decimalsB := p.decimals[indexIn], p.decimals[indexOut]
		var _reserveIn, _reserveOut uint256.Int
		_reserveIn.Div(_reserveIn.Mul(p.reserves[indexIn], big256.BONE), decimalsA)
		_reserveOut.Div(_reserveOut.Mul(p.reserves[indexOut], big256.BONE), decimalsB)

		var tmp uint256.Int
		amountOutScaled := tmp.Mul(amountOut, big256.BONE)
		amountOutScaled.Div(amountOutScaled, decimalsB)

		y, err := p._get_y(_reserveOut.Sub(&_reserveOut, amountOutScaled), xy, &_reserveIn)
		if err != nil {
			return nil, err
		}
		x := y.Sub(y, &_reserveIn)

		result := x.Div(x.Mul(x, decimalsA), big256.BONE)

		return big256.MulDivUp(result, result, p.feePrecision, tmp.Sub(p.feePrecision, p.fee)), nil
	}

	var tmp1, tmp2 uint256.Int
	tmp2.Mul(tmp1.Sub(p.reserves[indexOut], amountOut), tmp2.Sub(p.feePrecision, p.fee))
	return big256.MulDivUp(&tmp1, tmp1.Mul(p.reserves[indexIn], amountOut), p.feePrecision, &tmp2), nil
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

		if k.Lt(xy) {
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
					big256.U3,
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
