package tickerdualpool

// quoteGas is the hook's contribution; the v4 simulator adds its base gas.
// Local cold test-router estimates were 1.38-1.40M for one range and 2.43-2.44M
// for eight ranges, with one Aave vault. This estimate adds headroom and word
// crossings. It is a routing estimate, not a transaction gas limit or a proven
// worst-case bound. Actual Kyber executor and Monad billing still need checking.
func (h *Hook) quoteGas(q Quote) int64 {
	if h.SwapGas > 0 {
		return h.SwapGas
	}
	ranges := max(len(h.Pool.Buckets), 1)
	gas := int64(1450000 + 175000*(ranges-1))
	if h.TickSpacing > 0 {
		word := func(tick int32) int64 {
			compressed := int64(tick) / int64(h.TickSpacing)
			if tick < 0 && int64(tick)%int64(h.TickSpacing) != 0 {
				compressed--
			}
			return compressed >> 8
		}
		crossed := word(q.Swap.Tick) - word(h.Pool.Tick)
		if crossed < 0 {
			crossed = -crossed
		}
		gas += 25000 * crossed
	}
	// The measurements include one vault. A second vault adds another external
	// withdrawal/deposit path; retain extra allowance until separately measured.
	if h.Aave[0] != nil && h.Aave[1] != nil {
		gas += 500000
	}
	return gas
}
