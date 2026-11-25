package efficientprop

// Two fixed inputs against the model of the contracts (chain_model_test.go), on a listing shaped as the launch pool's wstETH/USDC one (18 and 6 decimals,
// a board unit of 1e12 wei of wstETH and of 1 base unit of USDC, the pool's bucket unit 1e12 wei of wstETH), and the
// gate's and the wire's small cases. Each test passes while the finding stands: it asserts what the chain does.

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// chCase is a wstETH/USDC listing at 3,600 USDC: the bid (wstETH in) three rungs to 0.05, 0.15 and 0.3 wstETH at 0,
// 2 % and 5 % under the anchor, the ask (USDC in) three rungs to 200, 500 and 1,000 USDC at 0, 300 and 800 ppm over
// it; a minimum fill of 10 USDC; a pool that bounds nothing.
func chCase() *chListing {
	l := &chListing{bd: 18, qd: 6, base0: true, class: "launch"}
	a := new(big.Int).Mul(big.NewInt(3600), chPow10(24))
	l.pair = new(big.Int).SetBit(new(big.Int), 8, 1)
	chOr(l.pair, 112, chN(12))
	depth := func(dExp uint64, nonce uint64, rungs [][2]uint64) *big.Int {
		w := big.NewInt(1)
		chOr(w, 12, chN(nonce))
		chOr(w, 60, chN(2_000_000_000))
		chOr(w, 112, chN(dExp))
		for j, rg := range rungs {
			chOr(w, 120+36*uint(j), chN(rg[0]|rg[1]<<16))
		}
		return w
	}
	board := func(nonce, filled, seq uint64) *big.Int {
		b := big.NewInt(1)
		chOr(b, 4, chN(nonce))
		chOr(b, 52, chN(filled))
		chOr(b, 92, new(big.Int).Sub(new(big.Int).Lsh(chOne, 96), chOne))
		chOr(b, 188, chN(chNow0-60_000))
		chOr(b, 236, chN(seq))
		return b
	}
	l.word = new(big.Int)
	chOr(l.word, 72, chN(chNow0-500))
	l.ctl = big.NewInt(1)
	l.s1 = new(big.Int)
	chOr(l.s1, 160, chN(10|6<<18))
	huge := new(big.Int).Lsh(chOne, 160)
	m96 := new(big.Int).Sub(new(big.Int).Lsh(chOne, 96), chOne)
	l.pool = &chPool{ideal: true, bandWad: new(big.Int),
		held: [2]*big.Int{new(big.Int).Set(huge), new(big.Int).Set(huge)}, owed: [2]*big.Int{new(big.Int), new(big.Int)},
		level: [2]*big.Int{new(big.Int).Set(huge), new(big.Int).Set(huge)},
		unit:  [2]*big.Int{chPow10(12), big.NewInt(1)}, capN: [2]*big.Int{new(big.Int), new(big.Int)}}
	side := func(bid bool, w0, b *big.Int) *chSide {
		return &chSide{bid: bid, w0: w0, w1: new(big.Int), board: b, word: l.word, ctl: l.ctl, pair: l.pair, s1: l.s1,
			a: a, P: new(big.Int), bRate: new(big.Int), bBurst: new(big.Int).Set(m96), pool: l.pool}
	}
	l.bid = side(true, depth(3, 77, [][2]uint64{{50, 0}, {150, 20000}, {300, 50000}}), board(77, 0, 7))
	l.ask = side(false, depth(6, 88, [][2]uint64{{200, 0}, {500, 300}, {1000, 800}}), board(88, 0, 7))
	return l
}

func chCaseSim(t *testing.T, l *chListing, blk chBlock) (*PoolSimulator, Extra) {
	t.Helper()
	p := chEntity(l)
	var extra Extra
	for range 2 {
		p, extra = chRefresh(&PoolTracker{}, l, p, []chBlock{blk}, true, chNewTally())
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	return sim, extra
}

// A word that refills the side at the route's second fill. The front refills a side where its word asks it,
// acknowledges the side's sequence and was stamped after the side's last fill (PropAMMWords.refills). A word that
// acknowledges the sequence after the next fill, stamped ahead of the block's clock, does not refill at the refresh
// nor at the first fill. It does at the second: the walk starts at rung 0 again and the desk's bucket cuts the fill
// short. The tracker gives such a direction no later fill (repeatOf).
func TestChainAWordRefillsTheSideAtTheSecondFill(t *testing.T) {
	l := chCase()
	// the bid's fill memory stands at 0.06 wstETH, inside rung 1; its bucket holds 600 USDC
	chOr(l.bid.board, 52, chN(60_000))
	l.bid.bBurst = big.NewInt(600e6)
	// the controls let a word refill; the word asks it of the bid, acknowledges sequence 8 (the board's is 7) and is
	// stamped 2 s ahead
	l.ctl.SetBit(l.ctl, 66, 1)
	l.word.SetInt64(0)
	chOr(l.word, 72, chN(chNow0+2000))
	l.word.SetBit(l.word, 232, 1)
	chOr(l.word, 234, chN(8))
	blk := chBlock{nowMs: chNow0}
	require.False(t, l.bid.refills(l.bid.board), "no refill at the refresh")
	sim, extra := chCaseSim(t, l, blk)
	pts := extra.Ladders[0]
	top := pts[len(pts)-1]

	first := new(big.Int).Mul(big.NewInt(5), chPow10(16)) // 0.05 wstETH
	rest := new(big.Int).Sub(top[0].ToBig(), first)
	chain := l.chain()
	q1, err := chCalc(sim, 0, first, nil)
	require.NoError(t, err)
	o1, cause := chain.bid.swap(first, blk)
	require.Empty(t, cause)
	require.LessOrEqual(t, q1.TokenAmountOut.Amount.Cmp(o1), 0)
	chApply(sim, 0, first, q1, nil)
	require.True(t, chain.bid.refills(chain.bid.board), "the word refills the side at the next fill")
	_, cause = chain.bid.swap(rest, blk)
	require.Equal(t, "NotEnoughDepth", cause, "the front refuses the second fill")
	_, err = chCalc(sim, 0, rest, nil)
	assert.ErrorIs(t, err, ErrSecondFill, "so the direction takes one fill a route")
	if extra.Repeat != nil {
		assert.Zero(t, extra.Repeat.Kind[0])
	}

	// a word that acknowledges the sequence at the refresh refills at the first fill or not at all: later fills stand
	l2 := chCase()
	l2.ctl.SetBit(l2.ctl, 66, 1)
	l2.word.SetInt64(0)
	chOr(l2.word, 72, chN(chNow0+2000))
	l2.word.SetBit(l2.word, 232, 1)
	chOr(l2.word, 234, chN(7))
	_, extra = chCaseSim(t, l2, blk)
	require.NotNil(t, extra.Repeat)
	assert.Equal(t, uint8(repeatOwn), extra.Repeat.Kind[0])
}
