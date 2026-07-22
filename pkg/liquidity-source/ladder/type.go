package ladder

// Point is one on-chain-probed sample of the curve: quoting AmountIn against
// the pool returns AmountOut. Like order-book's Level, amounts are stored as
// float64 (wei-scale, un-adjusted for decimals) rather than *uint256.Int:
// the underlying pools this package targets price against a live,
// continuously-drifting oracle feed, so a cached sample is never wei-exact
// anyway -- float64 keeps the serialized state compact without losing any
// precision that matters in practice. It's a bare 2-element array (not a
// named-field struct) so it serializes as a compact JSON array (["in","out"])
// instead of an object.
type Point [2]float64

func (p Point) AmountIn() float64  { return p[0] }
func (p Point) AmountOut() float64 { return p[1] }

// Extra is the serialized pool state shared by ladder-quoted pools: a
// sampled swap curve per swap direction (see Spline for how the samples are
// turned into a continuous quote), probed on-chain since the pricing formula
// itself isn't replicated off-chain. There is no separate
// paused/active flag: a paused pool (or a direction with no liquidity) is
// simply represented by an empty ladder for that direction, which
// QuoteAmountOut naturally rejects with ErrNoQuote.
type Extra struct {
	Ladders [2][]Point `json:"l"`
}

// PoolMeta is the default GetMetaInfo payload. Embedders that need extra
// static routing data (a contract address, a pair id, ...) should override
// GetMetaInfo entirely rather than extend this type.
type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
