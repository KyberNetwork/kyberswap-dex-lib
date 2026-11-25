package efficientprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// feeRefresh refreshes rungEthChain's listing 1 after set changed the chain and cfg the front's config, and gives the
// pool and the node.
func feeRefresh(t *testing.T, set func(*fakeChain), edit func(*Config)) (entity.Pool, *node) {
	t.Helper()
	chain := rungEthChain(false)
	if set != nil {
		set(chain)
	}
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	if edit != nil {
		edit(cfg)
	}
	p := ethListed(t, n, cfg)
	return ethTrack(t, n, cfg, p), n
}

// kinds are the later-fill kinds of the base-in and quote-in directions; zero where the state carries none.
func kinds(e Extra) [2]uint8 {
	if e.Repeat == nil {
		return [2]uint8{}
	}
	return e.Repeat.Kind
}

// The pool's fee is read from the chain at every head read. A pool that takes none keeps its later fills in both
// directions. A pool whose Factory's rates word is nonzero, or whose operator's share is, or whose fee cannot be read
// (a revert, a short answer, no Factory), keeps them in the quote-in direction alone: the fee comes off the quote's
// room, which the base-in direction pays from (FLAMMRoute._open).
func TestFeeReadWithdrawsTheBaseInLaterFills(t *testing.T) {
	cases := []struct {
		name string
		set  func(*fakeChain)
		fee  bool
	}{
		{"no fee", nil, false},
		{"rates", func(c *fakeChain) { c.feeRates = big.NewInt(500) }, true},
		{"notional rate of the correlated pairs", func(c *fakeChain) { c.feeRates = new(big.Int).Lsh(big.NewInt(7), 32) }, true},
		{"share", func(c *fakeChain) { c.feeShare = 1_500 }, true},
		{"rates read reverts", func(c *fakeChain) { c.feeBreak = "rates-revert" }, true},
		{"rates read short", func(c *fakeChain) { c.feeBreak = "rates-short" }, true},
		{"share read reverts", func(c *fakeChain) { c.feeBreak = "share-revert" }, true},
		{"share read short", func(c *fakeChain) { c.feeBreak = "share-short" }, true},
		{"factory read reverts", func(c *fakeChain) { c.feeBreak = "factory-revert" }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := feeRefresh(t, c.set, nil)
			k := kinds(extraOf(t, got))
			assert.NotZero(t, k[1], "the quote-in direction keeps its later fills")
			if c.fee {
				assert.Zero(t, k[0], "the base-in direction takes one fill a route")
			} else {
				assert.NotZero(t, k[0], "no fee: the base-in direction keeps its later fills")
			}
		})
	}
}

// The listing's Factory is read once, where it is scanned, and carried in the static extra; where the read failed it
// is empty.
func TestFactoryIsReadWhereTheListingIsScanned(t *testing.T) {
	got, _ := feeRefresh(t, nil, nil)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &se))
	assert.Equal(t, hexutil.Encode(testFeeFactory[:]), se.Factory)
	got, _ = feeRefresh(t, func(c *fakeChain) { c.feeBreak = "factory-revert" }, nil)
	se = StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(got.StaticExtra), &se))
	assert.Empty(t, se.Factory)
}

// The two reads ride in the head multicall: one spreadFeeRatesOf on the Factory and one operatorShare on the pool a
// refresh, and no multicall of their own.
func TestFeeReadsRideInTheHeadMulticall(t *testing.T) {
	count := func(n *node) (rates, share int) {
		for _, c := range n.calls() {
			if len(c.data) < 4 {
				continue
			}
			switch hexutil.Encode(c.data[:4]) {
			case hexutil.Encode(feeFactoryABI.Methods["spreadFeeRatesOf"].ID):
				if c.to == testFeeFactory {
					rates++
				}
			case hexutil.Encode(poolFeeABI.Methods["operatorShare"].ID):
				share++
			}
		}
		return
	}
	chain := rungEthChain(false)
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	st.set(slotAfter(n.time), 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, false)})
	cfg := ethConfig(st.URL)
	p := ethListed(t, n, cfg)
	r0, s0 := count(n)
	ethTrack(t, n, cfg, p)
	r1, s1 := count(n)
	assert.Equal(t, 1, r1-r0, "one rates read a refresh")
	assert.Equal(t, 1, s1-s0, "one share read a refresh")
}

// A listing named in the front's oneFillListings takes one fill in both directions, whatever the pool's fee.
func TestOneFillOverrideForcesBothDirections(t *testing.T) {
	got, _ := feeRefresh(t, nil, func(c *Config) { c.Fronts[0].OneFillListings = []uint32{1} })
	assert.Equal(t, [2]uint8{}, kinds(extraOf(t, got)))
}

// The read is the same on the head path (a front not read through /state).
func TestFeeReadOnTheHeadPath(t *testing.T) {
	for _, fee := range []bool{false, true} {
		chain, _ := rungChain()
		if fee {
			chain.feeShare = 100
		}
		n := chain.node(t, 1234)
		p := listed(t, n)
		k := kinds(extraOf(t, track(t, n, p)))
		assert.NotZero(t, k[1])
		assert.Equal(t, !fee, k[0] != 0, "fee %v", fee)
	}
	chain, _ := rungChain()
	chain.feeBreak = "rates-revert"
	n := chain.node(t, 1234)
	assert.Zero(t, kinds(extraOf(t, track(t, n, listed(t, n))))[0], "a rates read that reverts")
}
