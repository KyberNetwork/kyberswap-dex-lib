package ilyris

import (
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrMalformedExtra = errors.New("ilyris: pool extra could not be decoded")
	ErrEmptyBook      = errors.New("ilyris: pool has no populated bins")
)

// NewPoolSimulator builds a simulator from their stored pool entity.
//
// Registered with their factory by the PR that lands this package. Every failure returns an
// error rather than a half-built simulator: a pool that silently prices from an empty book
// quotes zero, and a zero is routed as a real offer of nothing.
func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var se StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &se); err != nil {
		return nil, ErrMalformedExtra
	}
	var ex Extra
	if err := json.Unmarshal([]byte(ep.Extra), &ex); err != nil {
		return nil, ErrMalformedExtra
	}
	if len(ep.Tokens) != 2 {
		return nil, ErrInvalidToken
	}
	if se.BinStepBps == 0 {
		// A zero bin step makes price(id) = 1 for every id -- every bin the same price, which
		// is not a pool, it is a rounding accident. Refuse rather than quote from it.
		return nil, ErrMalformedExtra
	}

	bins := make([]bin, 0, len(ex.Bins))
	sumX, sumY := new(big.Int), new(big.Int)
	for _, bj := range ex.Bins {
		x, y, ok := bj.reserves()
		if !ok {
			return nil, ErrMalformedExtra
		}
		if x.Sign() == 0 && y.Sign() == 0 {
			continue // empty bins carry no liquidity and only slow the traversal
		}
		bins = append(bins, bin{ID: bj.ID, ReserveX: x, ReserveY: y})
		sumX.Add(sumX, x)
		sumY.Add(sumY, y)
	}
	if len(bins) == 0 {
		return nil, ErrEmptyBook
	}
	// The traversal walks outward from the active bin and assumes ascending order. Sorting
	// here rather than trusting the tracker means a reordered payload cannot silently produce
	// a wrong quote -- it is cheap, and the failure it prevents is invisible.
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })

	// Lowercased here, once. Their PoolInfo.GetTokenIndex compares with ==, and addresses
	// reach us checksummed from our manifest and lowercased from their loader -- so folding
	// case at construction is what stops one of those two sources silently failing.
	tokens := []string{
		strings.ToLower(ep.Tokens[0].Address),
		strings.ToLower(ep.Tokens[1].Address),
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      tokens,
			Reserves:    []*big.Int{sumX, sumY},
			BlockNumber: ep.BlockNumber,
		}},
		binStepBps:       se.BinStepBps,
		activeID:         ex.ActiveID,
		decimalsX:        int(se.DecimalsX),
		decimalsY:        int(se.DecimalsY),
		bins:             bins,
		totalFeeRate:     ex.TotalFeeRate,
		guardSwapsPaused: ex.GuardSwapsPaused,
		guardFreezeEnd:   ex.GuardFreezeEnd,
		blockTimestamp:   ex.BlockTimestamp,
	}, nil
}
