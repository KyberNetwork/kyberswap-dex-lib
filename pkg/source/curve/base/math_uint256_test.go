package base

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func parityPool(numTokens int, aPrecision int64, initialA, futureA *big.Int, initialT, futureT int64) *PoolSimulator {
	return &PoolSimulator{
		InitialA:     initialA,
		FutureA:      futureA,
		InitialATime: initialT,
		FutureATime:  futureT,
		APrecision:   big.NewInt(aPrecision),
		numTokensBI:  big.NewInt(int64(numTokens)),
	}
}

// parityXPs returns golden and seeded-random xp vectors with balances up to 2^100.
func parityXPs(rng *rand.Rand) [][]*big.Int {
	var out = [][]*big.Int{
		{bignumber.NewBig10("101940884000000000000000000000000000000"), bignumber.NewBig10("107546110000000000000000000000000000000"), bignumber.NewBig10("208092128367874420986")},
		{bignumber.NewBig10("69265278000000000000000000000000000000"), bignumber.NewBig10("140296574000000000000000000000000000000")},
		{bignumber.NewBig10("100000000000000000"), bignumber.NewBig10("100000000000000000"), bignumber.NewBig10("1000000000000000000000000000000000000")},
	}
	for range 2000 {
		var n = 2 + rng.Intn(3)
		var xp = make([]*big.Int, n)
		for i := range xp {
			xp[i] = new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), uint(1+rng.Intn(100))))
			if xp[i].Sign() == 0 {
				xp[i].SetInt64(1)
			}
		}
		out = append(out, xp)
	}
	return out
}

func toU256s(xs []*big.Int) []*uint256.Int {
	var out = make([]*uint256.Int, len(xs))
	for i, x := range xs {
		out[i] = big256.FromBig(x)
	}
	return out
}

// TestGetDU256Parity checks exact match where both sides succeed. Where big.Int succeeds but
// an intermediate exceeds 2^256, the uint256 port must return errU256Overflow (on-chain revert).
func TestGetDU256Parity(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(1))
	var cells, matched, overflowBoundary int
	for _, xp := range parityXPs(rng) {
		for _, a := range []int64{1, 100, 150000, 1000000} {
			for _, aPrec := range []int64{1, 100} {
				if a < aPrec {
					continue
				}
				p := parityPool(len(xp), aPrec, big.NewInt(a), big.NewInt(a), 0, 0)
				wantD, wantErr := p.getD(xp, big.NewInt(a))
				gotD, gotErr := p.getDU256(toU256s(xp), uint256.NewInt(uint64(a)))
				cells++
				switch {
				case wantErr == nil && gotErr == nil:
					require.Equal(t, wantD.String(), gotD.Dec(), "xp=%v a=%d aPrec=%d", xp, a, aPrec)
					matched++
				case wantErr == nil:
					require.ErrorIs(t, gotErr, errU256Overflow, "xp=%v a=%d aPrec=%d", xp, a, aPrec)
					overflowBoundary++
				default:
					require.Error(t, gotErr, "big.Int errored, uint256 must too: xp=%v a=%d aPrec=%d", xp, a, aPrec)
				}
			}
		}
	}
	require.Positive(t, matched)
	t.Logf("getD grid: cells=%d matched=%d overflowBoundary=%d", cells, matched, overflowBoundary)
}

func TestAU256Parity(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(2))
	var cases int
	for range 5000 {
		// _A reads time.Now() itself; retry if the second ticks between the two reads.
		for {
			var now = time.Now().Unix()
			var initialT = now - rng.Int63n(1<<20)
			var futureT = now + 1 + rng.Int63n(1<<20)
			var p = parityPool(2, 100,
				big.NewInt(1+rng.Int63n(1000000)), big.NewInt(1+rng.Int63n(1000000)),
				initialT, futureT)
			var want = p._A()
			got, err := p._AU256(now)
			if now != time.Now().Unix() {
				continue
			}
			require.NoError(t, err)
			require.Equal(t, want.String(), got.Dec())
			break
		}
		cases++
	}
	require.Positive(t, cases)
}
