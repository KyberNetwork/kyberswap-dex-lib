package ilyris

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var _ pool.IPoolsListUpdater = (*PoolsListUpdater)(nil)

// defaultNewPoolLimit caps one scanning round. Their service calls GetNewPools repeatedly and
// carries our cursor between calls, so a small page is not a limit on total pools -- it just
// keeps any single round bounded.
const defaultNewPoolLimit = 100

// PoolsListUpdater walks BinFactory and reports pools their service has not seen.
type PoolsListUpdater struct {
	chain    chainReader
	factory  string
	exchange string
	limit    int
}

func NewPoolsListUpdater(chain chainReader, factory, exchange string) *PoolsListUpdater {
	if exchange == "" {
		exchange = string(DexType)
	}
	return &PoolsListUpdater{chain: chain, factory: factory, exchange: exchange, limit: defaultNewPoolLimit}
}

// GetNewPools resumes from the cursor in metadataBytes and returns the next page.
//
// The cursor is an OFFSET into the factory's append-only allPools array, which is the whole
// reason this is safe to resume: entries are only ever appended, so an offset that was valid
// last round is still valid this round and points at the same pool.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var md Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &md); err != nil {
			// A corrupt cursor must NOT silently restart the scan from zero: that re-emits
			// every pool as new on every round, forever. Surface it instead.
			return nil, metadataBytes, ErrMalformedExtra
		}
	}
	if md.Offset < 0 {
		return nil, metadataBytes, ErrMalformedExtra
	}

	addrs, total, err := u.chain.FactoryPools(ctx, u.factory, md.Offset, u.limit)
	if err != nil {
		// Hand the cursor back UNCHANGED. Advancing past pools we failed to read would skip
		// them permanently -- the array is append-only, so nothing ever revisits that range.
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(addrs))
	for _, a := range addrs {
		if a == "" || isZeroAddress(a) {
			continue
		}
		pools = append(pools, entity.Pool{
			Address:  strings.ToLower(a),
			Exchange: u.exchange,
			Type:     string(DexType),
			// Tokens, reserves and the book are deliberately left empty. The tracker fills
			// them from one lens call; duplicating that here would be a second code path
			// that can disagree with the first about what a pool is.
			Tokens: []*entity.PoolToken{{Swappable: true}, {Swappable: true}},
		})
	}

	next := md.Offset + len(addrs)
	if total > 0 && next > total {
		next = total
	}
	nextBytes, err := json.Marshal(Metadata{Offset: next})
	if err != nil {
		return pools, metadataBytes, err
	}
	return pools, nextBytes, nil
}
