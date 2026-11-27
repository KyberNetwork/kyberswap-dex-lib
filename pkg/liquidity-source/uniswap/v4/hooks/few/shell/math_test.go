package fewshell

import (
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func n(v uint64) *uint256.Int { return uint256.NewInt(v) }

// contractGetAmountIn is FewV2Math.getAmountIn as deployed, with its trailing +1.
func contractGetAmountIn(amountOut, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	var num, den uint256.Int
	num.Mul(reserveIn, amountOut).Mul(&num, n(1000))
	den.Sub(reserveOut, amountOut).Mul(&den, n(997))
	return num.Div(&num, &den).AddUint64(&num, 1)
}

// The pair's outputs for the four amounts the chain returned at block 26504066, before the protocol fee.
func TestGetAmountOut_OnChainNumbers(t *testing.T) {
	t.Parallel()
	rIn, rOut := u("16929650201529705921438"), u("29261029832819")
	for amountIn, want := range map[string]uint64{
		"1000000000000": 1723, "10000000000000": 17232, "100000000000000": 172320, "1000000000000000": 1723204,
	} {
		got, err := getAmountOut(n(uint64(big10(amountIn).Uint64())), &rIn, &rOut)
		require.NoError(t, err)
		assert.Equal(t, want, got.Uint64(), amountIn)
	}
}

func TestGetAmountOut_Errors(t *testing.T) {
	t.Parallel()
	r := n(1_000_000)
	_, err := getAmountOut(n(0), r, r)
	assert.ErrorIs(t, err, ErrInsufficientAmount)
	_, err = getAmountOut(n(1), n(0), r)
	assert.ErrorIs(t, err, ErrDegeneratePair)
	_, err = getAmountOut(n(1), r, n(0))
	assert.ErrorIs(t, err, ErrDegeneratePair)
	// 1 wei in against a deep pair rounds the output to 0 (FewV2Math reverts InsufficientAmount).
	_, err = getAmountOut(n(1), n(2_000_000), n(1_000_000))
	assert.ErrorIs(t, err, ErrInsufficientAmount)
}

// The protocol fee is ceil(amount*rate/1e6): a fee that rounds down would over-quote by a unit.
func TestProtocolFee_RoundsUp(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		amount uint64
		rate   uint32
		want   uint64
	}{
		{1723, 1000, 2}, {17232, 1000, 18}, {172320, 1000, 173}, {1723204, 1000, 1724}, // the chain's four fees at 10 bps
		{1000, 1000, 1}, {1001, 1000, 2}, {1, 1000, 1},
		{1723, 0, 0}, // the fee off
	} {
		assert.Equal(t, tc.want, protocolFee(n(tc.amount), tc.rate).Uint64(), "%d at %d ppm", tc.amount, tc.rate)
	}
}

// grossBeforeFee is the least amount that keeps net after the ceil fee.
func TestGrossBeforeFee_IsMinimal(t *testing.T) {
	t.Parallel()
	for _, rate := range []uint32{0, 25, 1000, 2500, 102375} {
		for range 2000 {
			net := n(1 + rand.Uint64N(1<<40))
			gross := grossBeforeFee(net, rate)
			after := func(g *uint256.Int) *uint256.Int { return new(uint256.Int).Sub(g, protocolFee(g, rate)) }
			assert.False(t, after(gross).Lt(net), "gross %s keeps net %s at %d ppm", gross, net, rate)
			assert.True(t, after(new(uint256.Int).SubUint64(gross, 1)).Lt(net), "gross-1 must fall short: %s %s %d", gross, net, rate)
		}
	}
}

// minAmountIn is the least input the pair needs, and the contract's getAmountIn is that or one more.
func TestMinAmountIn_IsMinimalAndNearTheContractsGetAmountIn(t *testing.T) {
	t.Parallel()
	for range 5000 {
		rIn, rOut := n(1_000+rand.Uint64N(1<<50)), n(1_000+rand.Uint64N(1<<50))
		out := n(1 + rand.Uint64N(rOut.Uint64()-1))
		in, err := minAmountIn(out, rIn, rOut)
		require.NoError(t, err)
		got, err := getAmountOut(in, rIn, rOut)
		require.NoError(t, err)
		assert.False(t, got.Lt(out), "in %s gives %s < %s (reserves %s %s)", in, got, out, rIn, rOut)
		if in.Cmp(n(1)) > 0 {
			less, err := getAmountOut(new(uint256.Int).SubUint64(in, 1), rIn, rOut)
			assert.True(t, err != nil || less.Lt(out), "in-1 must fall short (reserves %s %s out %s)", rIn, rOut, out)
		}
		contract := contractGetAmountIn(out, rIn, rOut)
		diff := new(uint256.Int).Sub(contract, in)
		assert.True(t, diff.IsZero() || diff.Eq(n(1)), "contract %s vs min %s", contract, in)
	}
}

// An exact division is where the contract's +1 over-supplies: reserves 1000/1000, out 1: (1000*1*1000)/(999*997) is not
// exact, so pick a case that is: rIn=997, rOut=1001, out=1 -> 997*1*1000 / (1000*997) = 1 exactly.
func TestMinAmountIn_ExactDivisionIsOneBelowTheContract(t *testing.T) {
	t.Parallel()
	in, err := minAmountIn(n(1), n(997), n(1001))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), in.Uint64())
	assert.Equal(t, uint64(2), contractGetAmountIn(n(1), n(997), n(1001)).Uint64())
	out, err := getAmountOut(in, n(997), n(1001))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), out.Uint64())
}

func TestMinAmountIn_Errors(t *testing.T) {
	t.Parallel()
	_, err := minAmountIn(n(0), n(10), n(10))
	assert.ErrorIs(t, err, ErrInsufficientAmount)
	_, err = minAmountIn(n(10), n(10), n(10)) // the whole reserve
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
	_, err = minAmountIn(n(1), n(0), n(10))
	assert.ErrorIs(t, err, ErrDegeneratePair)
}
