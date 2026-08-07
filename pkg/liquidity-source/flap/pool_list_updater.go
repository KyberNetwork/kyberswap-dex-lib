package flap

import (
	"context"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/flap/client"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poollist.RegisterFactoryC(DexType, NewPoolsListUpdater)

// Metadata tracks progress through the graduatinghot board's cursor pagination. That board is only
// usable for a one-shot bootstrap backfill: it is always sorted by volume24h_desc, so a token's
// position shifts between calls and cannot be used as a stable "give me only what's new" cursor across
// cycles. Once Done is true, GetNewPools stops making further requests.
type Metadata struct {
	Cursor string `json:"cursor"`
	Done   bool   `json:"done"`
}

type PoolsListUpdater struct {
	config      *Config
	boardClient *client.Client
}

func NewPoolsListUpdater(config *Config) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:      config,
		boardClient: client.NewClient(config.APIBaseURL, config.APIKey),
	}
}

// GetNewPools fetches one page of the board per call and turns each tradable (not yet graduated) entry
// into a pool. It never refetches a page once the whole board has been paged through once.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	if metadata.Done {
		return nil, metadataBytes, nil
	}

	board, err := u.boardClient.GetGraduatingHotBoard(ctx, metadata.Cursor)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(board.Items))
	for _, item := range board.Items {
		pool, ok := u.toPool(item)
		if !ok {
			continue
		}
		pools = append(pools, pool)
	}

	newMetadata := Metadata{Cursor: board.NextCursor}
	if board.NextCursor == "" {
		newMetadata.Done = true
	}
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("%s: discovered %d new pool(s), done=%v", DexType, len(pools), newMetadata.Done)

	return pools, newMetadataBytes, nil
}

// toPool converts one board item into an entity.Pool. `Listed` on this board means "listed on a DEX",
// i.e. already graduated off the bonding curve — curve-stage tokens are always Listed=false, so that's
// the flag we skip on, not the other way around. Graduated (progress == 100%) is kept as a defensive
// second check even though the graduatinghot board never reports it, in case a token crosses the
// threshold between the request and this pass.
func (u *PoolsListUpdater) toPool(item client.BoardItem) (entity.Pool, bool) {
	if item.Listed || item.Progress == graduationProgress {
		return entity.Pool{}, false
	}
	if item.Coin.Address == "" || item.QuoteToken == "" {
		return entity.Pool{}, false
	}

	quoteToken := valueobject.ZeroToWrappedLower(item.QuoteToken, u.config.ChainID)
	token := valueobject.WrapNativeLower(item.Coin.Address, u.config.ChainID)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		QuoteToken:    item.QuoteToken,
		PortalAddress: u.config.PortalAddress,
	})
	if err != nil {
		return entity.Pool{}, false
	}

	return entity.Pool{
		Address:     token,
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    []string{"0", "0"},
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: quoteToken, Swappable: true},
			{Address: token, Swappable: true},
		},
	}, true
}
