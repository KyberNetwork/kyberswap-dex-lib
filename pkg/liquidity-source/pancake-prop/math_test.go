package pancakeprop

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// livePair1 is the BSC USDT/WBNB price word at block 128674036, with the pair config at the
// time: token0 capacity 5000e18, token1 capacity 7.5e18.
const livePair1 = "0x2ee0bb89c4271020138800008ca0000000003e528f51f885b690001eada49401"

type quoteCase struct {
	Pk       string `json:"pk"`
	Fwd      bool   `json:"fwd"`
	MaxInput string `json:"maxInput"`
	Cum      string `json:"cum"`
	Amount   string `json:"amount"`
	Out      string `json:"out"`
	Err      string `json:"err"`
}

// quote_cases.json holds PropAMMPricing.quoteExactInput results from forks of BSC (random price
// words, capacities, cumulatives and sizes, edges of the capacity ceiling included) in both
// directions, the convex and concave curves, and every revert reachable from the price word.
func TestQuoteMatchesChain(t *testing.T) {
	raw, err := os.ReadFile("testdata/quote_cases.json")
	require.NoError(t, err)
	var cases []quoteCase
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases)

	for i, c := range cases {
		pr := unpackPrice(uint256.MustFromHex(c.Pk))
		out, err := pr.quote(c.Fwd, uint256.MustFromDecimal(c.MaxInput), uint256.MustFromDecimal(c.Cum),
			uint256.MustFromDecimal(c.Amount))
		if c.Err != "" {
			assert.Error(t, err, "case %d reverts with %s", i, c.Err)
			continue
		}
		require.NoError(t, err, "case %d", i)
		assert.Equal(t, c.Out, out.Dec(), "case %d", i)
	}
}

func TestUnpackPrice(t *testing.T) {
	p := unpackPrice(uint256.MustFromHex(livePair1))
	assert.Equal(t, uint64(128674085), p.expiry)
	assert.Equal(t, "1370961661057148518400", p.mid.Dec())
	assert.Equal(t, "180000000000000", p.base.Dec())
	assert.Equal(t, "3200000000000000", p.maxSpread.Dec())
	assert.False(t, p.concave)
	assert.Equal(t, uint64(2), p.growth)
	assert.Equal(t, [2]uint64{10000, 10000}, p.ratioMax)
	assert.Equal(t, [2]uint64{3000, 3000}, p.ratioFlat)
	assert.False(t, p.midErr)
}

// The flat region is linear in the input and the curve is strictly worse than the flat price, so
// a size crossing the flat boundary quotes below its linear extrapolation.
func TestQuoteDepthAwareSpread(t *testing.T) {
	p := unpackPrice(uint256.MustFromHex(livePair1))
	maxInput := uint256.MustFromDecimal("5000000000000000000000")
	var zero uint256.Int
	e18 := uint256.NewInt(1e18)
	at := func(usdt uint64) *uint256.Int {
		in := new(uint256.Int).Mul(uint256.NewInt(usdt), e18)
		out, err := p.quote(true, maxInput, &zero, in)
		require.NoError(t, err)
		return &out
	}
	assert.Equal(t, "1369727795562197", at(1).Dec()) // flat price, 9 bps below mid
	assert.Equal(t, "1369727795562197084", at(1000).Dec())
	assert.Equal(t, "4107282226587227294", at(3000).Dec()) // 1500 flat + 1500 on the curve
	linear := new(uint256.Int).Mul(at(1000), uint256.NewInt(3))
	assert.True(t, linear.Gt(at(3000)), "curve part is worse than the flat price")
}

func TestPowWad(t *testing.T) {
	var z uint256.Int
	// 0.428571428571428572 ** 1.2, matched against the pricing contract.
	require.NoError(t, powWad(&z, uint256.MustFromDecimal("428571428571428572"), uint256.MustFromDecimal("1200000000000000000")))
	assert.Equal(t, "361766091361761460", z.Dec())

	var l uint256.Int
	require.NoError(t, lnWad(&l, uint256.MustFromDecimal("428571428571428572")))
	assert.Equal(t, "-847297860387203613", l.ToBig().Sub(l.ToBig(), twoTo256()).String())

	assert.ErrorIs(t, lnWad(&l, uint256.NewInt(0)), ErrLnUndefined)
	require.NoError(t, expWad(&z, new(uint256.Int).Neg(mustU("41446531673892822313"))))
	assert.True(t, z.IsZero(), "exp below the representable range is zero")
	assert.ErrorIs(t, expWad(&z, mustU("135305999368893231589")), ErrExpOverflow)
}

func twoTo256() *big.Int { return new(big.Int).Lsh(big.NewInt(1), 256) }
