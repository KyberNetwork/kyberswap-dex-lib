package nemoprop

import (
	"errors"
	"math"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var errInvalidLadder = errors.New("invalid ladder")

// deliverable is how much of a token the vault can actually pay out: the
// smaller of its balance and its allowance to the proxy.
func deliverable(balance, allowance *uint256.Int) *uint256.Int {
	if allowance.Lt(balance) {
		return new(uint256.Int).Set(allowance)
	}
	return new(uint256.Int).Set(balance)
}

// shapeLadders turns a snapshot's raw (amountIn, amountOut) samples into the
// ladders quotes use. Direction 0 is
// base -> market, so its output is token 1; direction 1 the reverse.
func shapeLadders(raw [2][]ladder.Point, reserves [2]*uint256.Int, buffer int64) [2][]ladder.Point {
	return [2][]ladder.Point{
		shapeLadder(raw[0], reserves[1], buffer),
		shapeLadder(raw[1], reserves[0], buffer),
	}
}

// shapeLadder truncates raw where its output reaches the deliverable
// inventory, then applies the buffer. Truncation comes first: samples past
// the cap can't be delivered, and a flat run of them would read as a real
// rate. The first sample at or past the cap is replaced by the point where
// the chord from its predecessor reaches the cap. raw is never modified.
func shapeLadder(raw []ladder.Point, deliverableOut *uint256.Int, buffer int64) []ladder.Point {
	capOut := deliverableOut.Float64()
	if capOut <= 0 || len(raw) == 0 {
		return nil
	}

	shaped := make([]ladder.Point, 0, len(raw))
	var prevIn, prevOut float64
	for _, p := range raw {
		if p.AmountOut() >= capOut {
			if in := prevIn + (capOut-prevOut)*(p.AmountIn()-prevIn)/(p.AmountOut()-prevOut); in > prevIn {
				shaped = append(shaped, ladder.Point{in, capOut})
			}
			break
		}
		shaped = append(shaped, p)
		prevIn, prevOut = p.AmountIn(), p.AmountOut()
	}

	if buffer > 0 && buffer < bignumber.BasisPoint.Int64() {
		scale := float64(buffer) / float64(bignumber.BasisPoint.Int64())
		for i := range shaped {
			shaped[i][1] *= scale
		}
	}
	return shaped
}

// validateLadder rejects feed ladders the spline can't safely interpolate:
// amounts must be finite and positive, inputs strictly increasing and
// outputs non-decreasing.
func validateLadder(points []ladder.Point) error {
	var prevIn, prevOut float64
	for _, p := range points {
		in, out := p.AmountIn(), p.AmountOut()
		if math.IsNaN(in) || math.IsInf(in, 0) || math.IsNaN(out) || math.IsInf(out, 0) ||
			in <= prevIn || out <= 0 || out < prevOut {
			return errInvalidLadder
		}
		prevIn, prevOut = in, out
	}
	return nil
}

// newMarketState builds the immutable quoting state for one market from
// already-shaped ladders and deliverable reserves. It is built once per feed
// snapshot and shared by every quote on it, so callers must CloneState
// before UpdateBalance. Like every ladder tracker, it zeroes the output
// reserve of a direction with no quote: none of that token is obtainable.
func newMarketState(base, market string, reserves [2]*uint256.Int, ladders [2][]ladder.Point,
	blockNumber uint64) (*ladder.PoolSimulator, error) {
	extraBytes, err := json.Marshal(ladder.Extra{Ladders: ladders})
	if err != nil {
		return nil, err
	}
	state, err := ladder.NewPoolSimulator(entity.Pool{
		Type:        DexType,
		Reserves:    ladder.ZeroUnquotedReserves(entity.PoolReserves{reserves[0].Dec(), reserves[1].Dec()}, ladders),
		Tokens:      []*entity.PoolToken{{Address: base, Swappable: true}, {Address: market, Swappable: true}},
		Extra:       string(extraBytes),
		BlockNumber: blockNumber,
	})
	if err != nil {
		return nil, err
	}
	state.Gas = defaultGas
	return state, nil
}
