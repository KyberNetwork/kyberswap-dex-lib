package carbon

import (
	"fmt"
	"sort"

	"github.com/KyberNetwork/int256"
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

	if order.Y == nil || order.Z == nil {
		return u256.New0()
	}

	A := expandRate(order.A)
	B := expandRate(order.B)

	if A.Sign() == 0 && B.Sign() == 0 {
		return u256.New0()
	}

	if A.Sign() == 0 {
		result := new(uint256.Int).Mul(sourceAmount, B)
		result.MulDivOverflow(result, B, oneSquared)
		return result
	}

	x := sourceAmount
	y := order.Y
	z := order.Z

	temp1 := new(uint256.Int).Mul(z, uOne)

	temp2 := new(uint256.Int).Mul(A, y)
	temp2.Add(temp2, new(uint256.Int).Mul(B, z))

	temp3 := new(uint256.Int).Mul(temp2, x)

	numerator := new(uint256.Int).Mul(temp2, temp3)

	axTemp2 := new(uint256.Int).Mul(A, temp3)
	temp1Squared := new(uint256.Int).Mul(temp1, temp1)
	denominator := new(uint256.Int).Add(axTemp2, temp1Squared)

	if denominator.Sign() == 0 {
		return u256.New0()
	}

	result := new(uint256.Int).Div(numerator, denominator)

	return result
}

// tradeSourceAmount x * z^2 / ((A * y + B * z) * (A * y + B * z - A * x))
func tradeSourceAmount(targetAmount *uint256.Int, order *Order) *uint256.Int {
	if targetAmount == nil || targetAmount.Sign() <= 0 {
		return u256.New0()
	}

	if order.Y == nil || order.Z == nil {
		return u256.UMax.Clone()
	}

	A := expandRate(order.A)
	B := expandRate(order.B)

	if A.Sign() == 0 && B.Sign() == 0 {
		return u256.UMax.Clone()
	}

	if A.Sign() == 0 {
		result := new(uint256.Int).Mul(targetAmount, oneSquared)
		bSquared := new(uint256.Int).Mul(B, B)

		if bSquared.Sign() == 0 {
			return u256.UMax.Clone()
		}

		bSquaredMinus1 := new(uint256.Int).Sub(bSquared, u256.U1)
		result.Add(result, bSquaredMinus1)
		result.Div(result, bSquared)
		return result
	}

	x := targetAmount
	y := order.Y
	z := order.Z

	temp1 := new(uint256.Int).Mul(z, uOne)

	temp2 := new(uint256.Int).Mul(A, y)
	bz := new(uint256.Int).Mul(B, z)
	temp2.Add(temp2, bz)

	ax := new(uint256.Int).Mul(A, x)
	if !temp2.Gt(ax) {
		return u256.UMax.Clone()
	}

	temp3 := new(uint256.Int).Sub(temp2, ax)

	temp1.Mul(temp1, temp1)
	numerator := new(uint256.Int).Mul(x, temp1)

	denominator := new(uint256.Int).Mul(temp2, temp3)

	if denominator.Sign() == 0 {
		return u256.UMax.Clone()
	}

	denomMinus1 := new(uint256.Int).Sub(denominator, u256.U1)

	result := new(uint256.Int).Add(numerator, denomMinus1)
	result.Div(result, denominator)

	return result
}

func rateBySourceAmount(sourceAmount *uint256.Int, order *Order) Rate {
	input := new(uint256.Int).Set(sourceAmount)
	output := tradeTargetAmount(input, order)

	if output.Gt(order.Y) {
		input = tradeSourceAmount(order.Y, order)
		output = tradeTargetAmount(input, order)
		for output.Gt(order.Y) && input.Sign() > 0 {
			input.Sub(input, u256.U1)
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

func getParams(order *Order) (y, z, A, B *uint256.Int) {
	y = order.Y
	z = order.Z
	A = expandRate(order.A)
	B = expandRate(order.B)

	return
}

func getLimit(order *Order) *uint256.Int {
	y, z, A, B := getParams(order)

	if z.Sign() == 0 {
		return u256.New0()
	}

	result := new(uint256.Int).Mul(y, A)
	result.Add(result, new(uint256.Int).Mul(z, B))

	return result.Div(result, z)
}

func equalTargetAmount(order *Order, limit *uint256.Int) *uint256.Int {
	y, z, A, B := getParams(order)

	if A.Sign() == 0 {
		return y.Clone()
	}

	if B.Lt(limit) {
		return u256.New0()
	}

	bMinusLimit := new(uint256.Int).Sub(B, limit)
	res := new(uint256.Int).Mul(y, A)
	zBMinusLimit := new(uint256.Int).Mul(z, bMinusLimit)
	res.Add(res, zBMinusLimit)

	return res.Div(res, A)
}

func equalSourceAmount(order *Order, limit *uint256.Int) *uint256.Int {
	return tradeSourceAmount(equalTargetAmount(order, limit), order)
}

func sortByMinRate(x, y Rate) int {
	lhs := new(uint256.Int).Mul(x.Output, y.Input)
	rhs := new(uint256.Int).Mul(y.Output, x.Input)

	cmp := lhs.Cmp(rhs)
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
	remainingAmount := new(uint256.Int).Set(amount)

	for _, quote := range quotes {
		input := u256.Min(quote.Rate.Input, remainingAmount).Clone()
		order := ordersMap[quote.Id]
		output := trade(input, order).Output

		rate := Rate{Input: input, Output: output}
		if filter(rate) {
			actions = append(actions, &MatchAction{
				Id:     quote.Id,
				Input:  input,
				Output: output,
			})
			remainingAmount.Sub(remainingAmount, input)
			if remainingAmount.Sign() == 0 {
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
		fmt.Println("id", quote.Id, quote.Rate.Input.String(), quote.Rate.Output.String())
	}
	orders = append(orders, zeroOrder)

	var rates []Rate
	delta := int256.NewInt(0)

	for n := 1; n < len(orders); n++ {
		limit := getLimit(orders[n])
		rates = make([]Rate, n)
		total := u256.New0()

		for i := 0; i < n; i++ {
			equalAmt := equalize(orders[i], limit)
			rates[i] = trade(equalAmt, orders[i])
			total.Add(total, rates[i].Input)
		}

		delta.Sub(u256.SInt256(total), u256.SInt256(amount))

		if delta.Sign() == 0 {
			break
		}

		if delta.Sign() > 0 {
			lo := new(uint256.Int).Set(limit)
			hi := getLimit(orders[n-1])

			loPlus1 := new(uint256.Int).Add(lo, u256.U1)
			for loPlus1.Lt(hi) {
				limit = new(uint256.Int).Add(lo, hi)
				limit.Div(limit, u256.U2)

				rates = make([]Rate, n)
				total = u256.New0()

				for i := 0; i < n; i++ {
					equalAmt := equalize(orders[i], limit)
					rates[i] = trade(equalAmt, orders[i])
					total.Add(total, rates[i].Input)
				}

				delta.Sub(u256.SInt256(total), u256.SInt256(amount))

				if delta.Sign() == 0 {
					break
				}

				if delta.Sign() > 0 {
					lo.Set(limit)
				} else {
					hi.Set(limit)
				}

				loPlus1.Add(lo, u256.U1)
			}
			break
		}
	}

	if delta.Sign() > 0 {
		deltaAbs := (*uint256.Int)(delta)
		for i := len(rates) - 1; i >= 0 && deltaAbs.Sign() > 0; i-- {
			if rates[i].Input.Sign() == 0 {
				continue
			}

			newInput := u256.New0()
			if !rates[i].Input.Lt(deltaAbs) {
				newInput.Sub(rates[i].Input, deltaAbs)
			}

			rate := trade(newInput, orders[i])

			if !rate.Input.Gt(rates[i].Input) {
				actualReduction := new(uint256.Int).Sub(rates[i].Input, rate.Input)
				deltaAbs.Sub(deltaAbs, actualReduction)
			}

			rates[i] = rate
		}
	} else if delta.Sign() < 0 {
		deltaAbs := (*uint256.Int)(new(int256.Int).Neg(delta))
		for i := 0; i < len(rates) && deltaAbs.Sign() > 0; i++ {
			newInput := new(uint256.Int).Add(rates[i].Input, deltaAbs)
			rate := trade(newInput, orders[i])

			if rate.Input.Gt(rates[i].Input) {
				inputDiff := new(uint256.Int).Sub(rate.Input, rates[i].Input)
				if !inputDiff.Lt(deltaAbs) {
					break
				}
				deltaAbs.Sub(deltaAbs, inputDiff)
				rates[i] = rate
			}
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

func expandRate(rate uint64) *uint256.Int {
	if rate == 0 {
		return u256.New0()
	}

	res := uint256.NewInt(rate % one)
	return res.Lsh(res, uint(rate/one))
}
