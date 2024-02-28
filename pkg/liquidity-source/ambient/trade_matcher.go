package ambient

/* @notice Determines if we've terminated the swap execution. I.e. fully exhausted
 *         the specified swap quantity *OR* hit the directive's limit price. */
func hasSwapLeft(c *curveState, swap *swapDirective) bool {
	// bool inLimit = swap.isBuy_ ?
	// curve.priceRoot_ < swap.limitPrice_ :
	// curve.priceRoot_ > swap.limitPrice_;
	var inLimit bool
	if swap.isBuy {
		inLimit = c.priceRoot.Cmp(swap.limitPrice) == -1
	} else {
		inLimit = c.priceRoot.Cmp(swap.limitPrice) == 1
	}

	// return inLimit && (swap.qty_ > 0);
	return inLimit && (swap.qty.Cmp(big0) == 1)
}
