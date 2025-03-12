package swap

import (
	"github.com/holiman/uint256"
)

var zeroBI = uint256.NewInt(0)

func hasSellingY(limitOrder *LimitOrderPointU256) bool {
	if limitOrder == nil {
		return false
	}
	if limitOrder.SellingY == nil {
		return false
	}
	if limitOrder.SellingY.Cmp(zeroBI) == 0 {
		return false
	}
	return true
}

func hasSellingX(limitOrder *LimitOrderPointU256) bool {
	if limitOrder == nil {
		return false
	}
	if limitOrder.SellingX == nil {
		return false
	}
	if limitOrder.SellingX.Cmp(zeroBI) == 0 {
		return false
	}
	return true
}

func InitY2X(liquidities []LiquidityPointU256, limitOrders []LimitOrderPointU256, currentPoint int) OrderData {
	var orderData OrderData
	orderData.Liquidities = liquidities
	orderData.LimitOrders = limitOrders
	idx := 0
	for idx < len(liquidities) && liquidities[idx].Point <= currentPoint {
		idx++
	}
	orderData.LiquidityIdx = idx
	idx = 0
	for idx < len(limitOrders) && (limitOrders[idx].Point < currentPoint || !hasSellingX(&limitOrders[idx])) {
		idx++
	}
	orderData.LimitOrderIdx = idx
	return orderData
}

func InitX2Y(liquidities []LiquidityPointU256, limitOrders []LimitOrderPointU256, currentPoint int) OrderData {
	var orderData OrderData
	orderData.Liquidities = liquidities
	orderData.LimitOrders = limitOrders
	idx := len(liquidities) - 1
	for idx >= 0 && liquidities[idx].Point > currentPoint {
		idx--
	}
	orderData.LiquidityIdx = idx
	idx = len(limitOrders) - 1
	for idx >= 0 && (limitOrders[idx].Point > currentPoint || !hasSellingY(&limitOrders[idx])) {
		idx--
	}
	orderData.LimitOrderIdx = idx
	return orderData
}

func (orderData *OrderData) findRightPoint(rightBoundary int) int {
	rightPoint := rightBoundary
	if orderData.LiquidityIdx < len(orderData.Liquidities) {
		if rightPoint > orderData.Liquidities[orderData.LiquidityIdx].Point {
			rightPoint = orderData.Liquidities[orderData.LiquidityIdx].Point
		}
	}
	if orderData.LimitOrderIdx < len(orderData.LimitOrders) {
		if rightPoint > orderData.LimitOrders[orderData.LimitOrderIdx].Point {
			rightPoint = orderData.LimitOrders[orderData.LimitOrderIdx].Point
		}
	}
	return rightPoint
}

func (orderData *OrderData) MoveY2X(point, pointDelta int) int {
	mapPt := point / pointDelta
	if point < 0 && point%pointDelta != 0 {
		mapPt-- // round towards negative infinity
	}
	mapPt += 1
	bitIdx := (mapPt%256 + 256) % 256
	rightBoundary := (mapPt + 255 - bitIdx) * pointDelta

	idx := orderData.LiquidityIdx
	for idx < len(orderData.Liquidities) && orderData.Liquidities[idx].Point <= point {
		idx++
	}
	orderData.LiquidityIdx = idx

	idx = orderData.LimitOrderIdx
	for idx < len(orderData.LimitOrders) && (orderData.LimitOrders[idx].Point < point || !hasSellingX(&orderData.LimitOrders[idx])) {
		idx++
	}
	orderData.LimitOrderIdx = idx
	return orderData.findRightPoint(rightBoundary)
}

func (orderData *OrderData) findLeftPoint(leftBoundary int) int {
	leftPoint := leftBoundary
	if orderData.LiquidityIdx >= 0 {
		if leftPoint < orderData.Liquidities[orderData.LiquidityIdx].Point {
			leftPoint = orderData.Liquidities[orderData.LiquidityIdx].Point
		}
	}
	if orderData.LimitOrderIdx >= 0 {
		if leftPoint < orderData.LimitOrders[orderData.LimitOrderIdx].Point {
			leftPoint = orderData.LimitOrders[orderData.LimitOrderIdx].Point
		}
	}
	return leftPoint
}

func (orderData *OrderData) MoveX2Y(point, pointDelta int) int {
	mapPt := point / pointDelta
	if point < 0 && point%pointDelta != 0 {
		mapPt-- // round towards negative infinity
	}
	bitIdx := (mapPt%256 + 256) % 256
	leftBoundary := (mapPt - bitIdx) * pointDelta

	idx := orderData.LiquidityIdx
	for idx >= 0 && orderData.Liquidities[idx].Point > point {
		idx--
	}
	orderData.LiquidityIdx = idx

	idx = orderData.LimitOrderIdx
	for idx >= 0 &&
		(orderData.LimitOrders[idx].Point > point || !hasSellingY(&orderData.LimitOrders[idx])) {
		idx--
	}
	orderData.LimitOrderIdx = idx
	return orderData.findLeftPoint(leftBoundary)
}

func (orderData *OrderData) ConsumeLimitOrder(isY2X bool) {
	if isY2X {
		if orderData.LimitOrderIdx < len(orderData.LimitOrders) {
			orderData.LimitOrderIdx++
		}
	} else {
		if orderData.LimitOrderIdx >= 0 {
			orderData.LimitOrderIdx--
		}
	}
}
