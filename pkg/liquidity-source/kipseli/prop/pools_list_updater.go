package prop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools lists one pool per listed token against the venue's quote
// token, from the quoter the swap path actually uses for Config.Dest. It
// returns the full list every poll: the list is a handful of tokens, the
// caller skips pools it already has, and a pool record lost downstream gets
// re-created instead of staying behind a pagination cursor.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	router := common.HexToAddress(u.cfg.RouterAddress)
	snap, err := fetchSnapshot(ctx, u.ethrpcClient, router, u.cfg.dest(), nil, nil, titan.State{})
	if err != nil {
		logger.WithFields(logger.Fields{"dexId": u.cfg.DexID, "error": err}).Error("kipseli-prop: lens call failed")
		return nil, metadataBytes, err
	}
	if snap.QuoteToken == (common.Address{}) || snap.Quoter == (common.Address{}) {
		return nil, metadataBytes, ErrNoQuoter
	}

	staticExtraBytes, _ := json.Marshal(StaticExtra{RouterAddress: strings.ToLower(u.cfg.RouterAddress)})
	quoteToken := hexutil.Encode(snap.QuoteToken[:])
	now := time.Now().Unix()

	pools := make([]entity.Pool, 0, len(snap.ListedTokens))
	for _, token := range snap.ListedTokens {
		if token == snap.QuoteToken {
			continue
		}
		pools = append(pools, entity.Pool{
			Address:   syntheticPoolAddress(u.cfg.DexID, token, snap.QuoteToken),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token[:]), Swappable: true},
				{Address: quoteToken, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	logger.WithFields(logger.Fields{"dexId": u.cfg.DexID, "quoter": snap.Quoter, "swapImpl": snap.SwapImpl}).
		Infof("kipseli-prop: listed %d pools", len(pools))
	return pools, metadataBytes, nil
}

func syntheticPoolAddress(dexID string, token, quote common.Address) string {
	return strings.ToLower(dexID) + "_" + hexutil.Encode(token[:]) + "_" + hexutil.Encode(quote[:])
}
