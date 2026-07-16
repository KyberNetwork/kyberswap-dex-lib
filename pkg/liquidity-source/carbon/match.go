package carbon

import (
	"sort"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type MatchType int

const (
	MatchTypeFast MatchType = iota + 1
	MatchTypeBest
)

type Rate struct {
	Input  *uint256.Int
	Output *uint256.Int
}

type Quote struct {
	Id   string
	Rate Rate
}

type MatchAction struct {
	Id     string
	Input  *uint256.Int
	Output *uint256.Int
}

type MatchOptions struct {
	Fast []*MatchAction
	Best []*MatchAction
}

type EncodedOrderMap map[string]*Order

type Filter func(rate Rate) bool

func defaultFilter(rate Rate) bool {
	return rate.Input != nil && rate.Input.Sign() > 0 &&
		rate.Output != nil && rate.Output.Sign() > 0
}

// tradeTargetAmount x * (A * y + B * z)^2 / (A * x * (A * y + B * z) + z^2)
func tradeTargetAmount(sourceAmount *uint256.Int, order *Order) *uint256.Int {
	if sourceAmount == nil || sourceAmount.Sign() <= 0 {
		return u256.New0()
	}
	if order.Y.Sign() == 0 {
		return u256.New0()
	}

	var A, B uint256.Int
	expandRate(&A, order.A)
	expandRate(&B, order.B)

	if A.IsZero() && B.IsZero() {
		return u256.New0()
	}

	x := sourceAmount
	y := order.Y
	z := order.Z

	if A.IsZero() {
		var bSq, res uint256.Int
		bSq.Mul(&B, &B)
		return u256.MulDivDown(&res, x, &bSq, oneSquared)
	}

	var temp1 uint256.Int
	temp1.Mul(z, uOne)

	var temp2, zB uint256.Int
	temp2.Mul(&A, y)
	zB.Mul(&B, z)
	temp2.Add(&temp2, &zB)

	var temp3 uint256.Int
	temp3.Mul(&temp2, x)

	var mf1, mf2 uint256.Int
	factor := u256.Max(minFactor(&mf1, &temp1, &temp1), minFactor(&mf2, &temp3, &A))
	if factor.IsZero() {
		factor = u256.U1
	}

	var temp4, temp5 uint256.Int
	u256.MulDivUp(&temp4, &temp1, &temp1, factor)
	u256.MulDivUp(&temp5, &temp3, &A, factor)
	temp5.Add(&temp4, &temp5)

	if !temp5.Lt(&temp4) {
		var div, res uint256.Int
		div.Div(&temp3, factor)
		return u256.MulDivDown(&res, &temp2, &div, &temp5)
	}

	u256.MulDivUp(&temp1, &temp1, &temp1, &temp3)
	temp1.Add(&A, &temp1)
	return temp2.Div(&temp2, &temp1)
}

// tradeSourceAmount x * z^2 / ((A * y + B * z) * (A * y + B * z - A * x))
func tradeSourceAmount(targetAmount *uint256.Int, order *Order) *uint256.Int {
	if targetAmount == nil || targetAmount.Sign() <= 0 {
		return u256.New0()
	}
	if order.Y.Sign() == 0 {
		return u256.UMax.Clone()
	}

	var A, B uint256.Int
	expandRate(&A, order.A)
	expandRate(&B, order.B)

	if A.IsZero() && B.IsZero() {
		return u256.UMax.Clone()
	}

	x := targetAmount
	y := order.Y
	z := order.Z

	if A.IsZero() {
		var bSq uint256.Int
		bSq.Mul(&B, &B)
		if bSq.IsZero() {
			return u256.UMax.Clone()
		}
		var res uint256.Int
		return u256.MulDivUp(&res, x, oneSquared, &bSq)
	}

	var temp1 uint256.Int
	temp1.Mul(z, uOne)

	var temp2, zB uint256.Int
	temp2.Mul(y, &A)
	zB.Mul(z, &B)
	temp2.Add(&temp2, &zB)

	var ax uint256.Int
	ax.Mul(x, &A)
	if !temp2.Gt(&ax) {
		return u256.UMax.Clone()
	}

	var temp3 uint256.Int
	temp3.Sub(&temp2, &ax)

	var mf1, mf2 uint256.Int
	factor := u256.Max(minFactor(&mf1, &temp1, &temp1), minFactor(&mf2, &temp2, &temp3))
	if factor.IsZero() {
		factor = u256.U1
	}

	var temp4, temp5 uint256.Int
	u256.MulDivUp(&temp4, &temp1, &temp1, factor)
	u256.MulDivDown(&temp5, &temp2, &temp3, factor)

	if temp5.IsZero() {
		return u256.UMax.Clone()
	}

	var res uint256.Int
	return u256.MulDivUp(&res, x, &temp4, &temp5)
}

func rateBySourceAmount(sourceAmount *uint256.Int, order *Order) Rate {
	input := new(uint256.Int).Set(sourceAmount)
	output := tradeTargetAmount(input, order)

	if output.Gt(order.Y) {
		input = tradeSourceAmount(order.Y, order)
		output = tradeTargetAmount(input, order)
		for output.Gt(order.Y) && input.Sign() > 0 {
			input.SubUint64(input, 1)
			output = tradeTargetAmount(input, order)
		}
	}

	return Rate{Input: input, Output: output}
}

func rateByTargetAmount(targetAmount *uint256.Int, order *Order) Rate {
	input := u256.Min(targetAmount, order.Y).Clone()
	output := tradeSourceAmount(input, order)

	return Rate{Input: input, Output: output}
}

func getParams(A, B *uint256.Int, order *Order) (y, z *uint256.Int) {
	y = order.Y
	z = order.Z
	expandRate(A, order.A)
	expandRate(B, order.B)
	return
}

func getLimit(dst *uint256.Int, order *Order) *uint256.Int {
	var A, B uint256.Int
	y, z := getParams(&A, &B, order)

	if z.IsZero() {
		return dst.Clear()
	}

	dst.Mul(y, &A)
	var zB uint256.Int
	zB.Mul(z, &B)
	dst.Add(dst, &zB)
	return dst.Div(dst, z)
}

func equalTargetAmount(order *Order, limit *uint256.Int) *uint256.Int {
	var A, B uint256.Int
	y, z := getParams(&A, &B, order)

	if A.IsZero() {
		return y.Clone()
	}

	// num = A*y + B*z = getLimit(order) * z
	var num, limitZ uint256.Int
	num.Mul(y, &A)
	var Bz uint256.Int
	Bz.Mul(z, &B)
	num.Add(&num, &Bz)

	limitZ.Mul(z, limit)
	if !num.Gt(&limitZ) {
		return u256.New0()
	}

	// (A*y + B*z - limit*z) / A — safe since num > limitZ
	var res uint256.Int
	res.Sub(&num, &limitZ)
	return res.Div(&res, &A)
}

func equalSourceAmount(order *Order, limit *uint256.Int) *uint256.Int {
	return tradeSourceAmount(equalTargetAmount(order, limit), order)
}

func sortByMinRate(x, y Rate) int {
	var lhs, rhs uint256.Int
	lhs.Mul(x.Output, y.Input)
	rhs.Mul(y.Output, x.Input)

	cmp := lhs.Cmp(&rhs)
	if cmp != 0 {
		return cmp
	}

	return x.Output.Cmp(y.Output)
}

func sortByMaxRate(x, y Rate) int {
	return sortByMinRate(y, x)
}

func sortedQuotes(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	trade func(*uint256.Int, *Order) Rate,
	sortFn func(Rate, Rate) int,
) []Quote {
	quotes := make([]Quote, 0, len(ordersMap))

	for idStr, order := range ordersMap {
		rate := trade(amount, order)
		quotes = append(quotes, Quote{Id: idStr, Rate: rate})
	}

	sort.Slice(quotes, func(i, j int) bool {
		return sortFn(quotes[i].Rate, quotes[j].Rate) > 0
	})

	return quotes
}

func matchFast(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	quotes []Quote,
	filter Filter,
	trade func(*uint256.Int, *Order) Rate,
) []*MatchAction {
	actions := make([]*MatchAction, 0)
	var remaining uint256.Int
	remaining.Set(amount)

	for _, quote := range quotes {
		input := u256.Min(quote.Rate.Input, &remaining).Clone()
		order := ordersMap[quote.Id]
		output := trade(input, order).Output

		rate := Rate{Input: input, Output: output}
		if filter(rate) {
			actions = append(actions, &MatchAction{
				Id:     quote.Id,
				Input:  input,
				Output: output,
			})
			remaining.Sub(&remaining, input)
			if remaining.IsZero() {
				break
			}
		}
	}

	return actions
}

func matchBest(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	quotes []Quote,
	filter Filter,
	trade func(*uint256.Int, *Order) Rate,
	equalize func(*Order, *uint256.Int) *uint256.Int,
) []*MatchAction {
	if len(quotes) == 0 {
		return []*MatchAction{}
	}

	zeroOrder := &Order{
		Y: u256.New0(),
		Z: u256.New0(),
		A: 0,
		B: 0,
	}

	orders := make([]*Order, 0, len(quotes)+1)
	for _, quote := range quotes {
		orders = append(orders, ordersMap[quote.Id])
	}
	orders = append(orders, zeroOrder)

	var rates []Rate
	var delta uint256.Int
	deltaSign := 0

	for n := 1; n < len(orders); n++ {
		var limit uint256.Int
		getLimit(&limit, orders[n])
		rates = make([]Rate, n)
		var total uint256.Int

		for i := 0; i < n; i++ {
			equalAmt := equalize(orders[i], &limit)
			rates[i] = trade(equalAmt, orders[i])
			total.Add(&total, rates[i].Input)
		}

		if total.Gt(amount) {
			delta.Sub(&total, amount)
			deltaSign = 1
		} else if total.Lt(amount) {
			delta.Sub(amount, &total)
			deltaSign = -1
		} else {
			delta.Clear()
			deltaSign = 0
		}

		if deltaSign == 0 {
			break
		}

		if deltaSign > 0 {
			var lo uint256.Int
			lo.Set(&limit)
			var hi uint256.Int
			getLimit(&hi, orders[n-1])

			var loPlus1 uint256.Int
			loPlus1.AddUint64(&lo, 1)
			for loPlus1.Lt(&hi) {
				limit.Add(&lo, &hi)
				limit.Div(&limit, u256.U2)

				rates = make([]Rate, n)
				total.Clear()

				for i := 0; i < n; i++ {
					equalAmt := equalize(orders[i], &limit)
					rates[i] = trade(equalAmt, orders[i])
					total.Add(&total, rates[i].Input)
				}

				if total.Gt(amount) {
					delta.Sub(&total, amount)
					deltaSign = 1
				} else if total.Lt(amount) {
					delta.Sub(amount, &total)
					deltaSign = -1
				} else {
					delta.Clear()
					deltaSign = 0
				}

				if deltaSign == 0 {
					break
				}

				if deltaSign > 0 {
					lo.Set(&limit)
				} else {
					hi.Set(&limit)
				}

				loPlus1.AddUint64(&lo, 1)
			}
			break
		}
	}

	if deltaSign > 0 {
		for i := len(rates) - 1; i >= 0 && delta.Sign() > 0; i-- {
			if rates[i].Input.Sign() == 0 {
				continue
			}

			var newInput uint256.Int
			if !rates[i].Input.Lt(&delta) {
				newInput.Sub(rates[i].Input, &delta)
			}

			rate := trade(&newInput, orders[i])

			if !rate.Input.Gt(rates[i].Input) {
				var actualReduction uint256.Int
				actualReduction.Sub(rates[i].Input, rate.Input)
				delta.Sub(&delta, &actualReduction)
			}

			rates[i] = rate
		}
	} else if deltaSign < 0 {
		for i := 0; i < len(rates); i++ {
			var newInput uint256.Int
			newInput.Add(rates[i].Input, &delta)
			rate := trade(&newInput, orders[i])

			var inputDiff uint256.Int
			inputDiff.Sub(rate.Input, rates[i].Input)

			if !inputDiff.Lt(&delta) {
				break
			}

			delta.Sub(&delta, &inputDiff)
			rates[i] = rate
		}
	}

	actions := make([]*MatchAction, 0)
	for i, rate := range rates {
		if filter(rate) {
			actions = append(actions, &MatchAction{
				Id:     quotes[i].Id,
				Input:  rate.Input,
				Output: rate.Output,
			})
		}
	}

	return actions
}

func matchBy(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	matchTypes []MatchType,
	filter Filter,
	trade func(*uint256.Int, *Order) Rate,
	sortFn func(Rate, Rate) int,
	equalize func(*Order, *uint256.Int) *uint256.Int,
) *MatchOptions {
	quotes := sortedQuotes(amount, ordersMap, trade, sortFn)
	res := &MatchOptions{}
	for _, matchType := range matchTypes {
		switch matchType {
		case MatchTypeFast:
			res.Fast = matchFast(amount, ordersMap, quotes, filter, trade)
		case MatchTypeBest:
			res.Best = matchBest(amount, ordersMap, quotes, filter, trade, equalize)
		}
	}

	return res
}

func MatchBySourceAmount(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	matchTypes []MatchType,
	filter Filter,
) *MatchOptions {
	if filter == nil {
		filter = defaultFilter
	}

	return matchBy(amount, ordersMap, matchTypes, filter, rateBySourceAmount, sortByMinRate, equalSourceAmount)
}

func MatchByTargetAmount(
	amount *uint256.Int,
	ordersMap EncodedOrderMap,
	matchTypes []MatchType,
	filter Filter,
) *MatchOptions {
	if filter == nil {
		filter = defaultFilter
	}

	return matchBy(amount, ordersMap, matchTypes, filter, rateByTargetAmount, sortByMaxRate, equalTargetAmount)
}

// expandRate writes the expanded value of rate into dst and returns dst.
func expandRate(dst *uint256.Int, rate uint64) *uint256.Int {
	if rate == 0 {
		return dst.Clear()
	}
	dst.SetUint64(rate % one)
	return dst.Lsh(dst, uint(rate/one))
}

// mul512 returns the full 512-bit product of x*y as (hi, lo) by value.
// Uses 128-bit halving to compute hi without MulMod, avoiding the Reciprocal cost.
func mul512(x, y *uint256.Int) (hi, lo uint256.Int) {
	lo.Mul(x, y)

	var xL, xH, yL, yH uint256.Int
	xL.And(x, u256.UMaxU128)
	xH.Rsh(x, 128)
	yL.And(y, u256.UMaxU128)
	yH.Rsh(y, 128)

	// cross = (xL*yL)>>128 + xH*yL + xL*yH
	// Track 256-bit overflow carries: each carry contributes 2^128 to hi.
	var cross, tmp uint256.Int
	cross.Mul(&xL, &yL)
	cross.Rsh(&cross, 128)

	tmp.Mul(&xH, &yL)
	_, c1 := cross.AddOverflow(&cross, &tmp)

	tmp.Mul(&xL, &yH)
	_, c2 := cross.AddOverflow(&cross, &tmp)

	hi.Mul(&xH, &yH)
	cross.Rsh(&cross, 128)
	hi.Add(&hi, &cross)
	if c1 {
		hi.Add(&hi, u256.U2Pow128)
	}
	if c2 {
		hi.Add(&hi, u256.U2Pow128)
	}
	return
}

// minFactor writes the result into dst and returns dst.
func minFactor(dst, x, y *uint256.Int) *uint256.Int {
	hi, lo := mul512(x, y)
	var notLo uint256.Int
	notLo.Not(&lo)
	if hi.Gt(&notLo) {
		return dst.AddUint64(&hi, 2)
	}
	return dst.AddUint64(&hi, 1)
}
