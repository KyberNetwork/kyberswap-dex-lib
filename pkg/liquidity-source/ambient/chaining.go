package ambient

import "math/big"

/* @notice Increments a PairFlow accumulator with the flows from a swap leg.
* @param flow The PairFlow object being accumulated. Function writes to this
*   structure.
* @param inBaseQty Whether the swap was denominated in base or quote side tokens.
* @param base The base side token flows. Negative when going to the user, positive
*   for flows going to the pool.
* @param quote The quote side token flows. Negative when going to the user, positive
*   for flows going to the pool.
* @param proto The amount of protocol fees collected by the swap operation. (The
*   total flows must be inclusive of this value). */
func (f *pairFlow) accumSwap(inBaseQty bool, base *big.Int, quote *big.Int, proto *big.Int) {
	// accumFlow(flow, base, quote);
	f.accumFlow(base, quote)

	// if (inBaseQty) {
	// 		flow.quoteProto_ += proto;
	// } else {
	// 		flow.baseProto_ += proto;
	// }
	if inBaseQty {
		f.quoteProto.Add(f.quoteProto, proto)
	} else {
		f.baseProto.Add(f.baseProto, proto)
	}
}

/* @notice Increments a PairFlow accumulator with a set of pre-determined flows.
* @param flow The PairFlow object being accumulated. Function writes to this
*   structure.
* @param base The base side token flows. Negative when going to the user, positive
*   for flows going to the pool.
* @param quote The quote side token flows. Negative when going to the user, positive
*   for flows going to the pool. */
func (f *pairFlow) accumFlow(base *big.Int, quote *big.Int) {
	// 	 flow.baseFlow_ += base;
	f.baseFlow.Add(f.baseFlow, base)

	// 	 flow.quoteFlow_ += quote;
	f.quoteFlow.Add(f.quoteFlow, quote)
}
