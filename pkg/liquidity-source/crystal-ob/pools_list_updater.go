package crystalob

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, ethrpcClient: ethrpcClient}
}

// GetNewPools lists swappable markets from Crystal.allMarkets. Launchpad markets still on their
// bonding curve map to a placeholder in getMarketByTokens; they are kept in metadata and
// rechecked every run, so the tracker only polls markets swap() accepts.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var length *big.Int
	var weth common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: crystalABI, Target: u.config.RouterAddress, Method: "allMarketsLength"},
			[]any{&length}).
		AddCall(&ethrpc.Call{ABI: crystalABI, Target: u.config.RouterAddress, Method: "weth"}, []any{&weth}).
		Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	end := min(int(length.Int64()), metadata.Offset+listBatchSize)
	newMarkets := make([]common.Address, max(end-metadata.Offset, 0))
	if err := u.aggregate(ctx, len(newMarkets), func(req *ethrpc.Request, i int) {
		req.AddCall(&ethrpc.Call{ABI: crystalABI, Target: u.config.RouterAddress, Method: "allMarkets",
			Params: []any{big.NewInt(int64(metadata.Offset + i))}}, []any{&newMarkets[i]})
	}); err != nil {
		return nil, metadataBytes, err
	}

	markets := append(metadata.Pending, newMarkets...)
	infos := make([]struct{ Info MarketInfoRPC }, len(markets))
	if err := u.aggregate(ctx, len(markets), func(req *ethrpc.Request, i int) {
		req.AddCall(&ethrpc.Call{ABI: crystalABI, Target: u.config.RouterAddress, Method: "getMarket",
			Params: []any{markets[i]}}, []any{&infos[i]})
	}); err != nil {
		return nil, metadataBytes, err
	}
	canonical := make([]common.Address, len(markets))
	if err := u.aggregate(ctx, len(markets), func(req *ethrpc.Request, i int) {
		req.AddCall(&ethrpc.Call{ABI: crystalABI, Target: u.config.RouterAddress, Method: "getMarketByTokens",
			Params: []any{infos[i].Info.QuoteAsset, infos[i].Info.BaseAsset}}, []any{&canonical[i]})
	}); err != nil {
		return nil, metadataBytes, err
	}

	var pools []entity.Pool
	var pending []common.Address
	router := hexutil.Encode(common.HexToAddress(u.config.RouterAddress).Bytes())
	for i, market := range markets {
		if canonical[i] != market {
			pending = append(pending, market)
			continue
		}
		m := &infos[i].Info
		staticExtra, _ := json.Marshal(StaticExtra{
			MarketType:  m.MarketType.Uint64(),
			ScaleFactor: uint256.MustFromBig(m.ScaleFactor),
			TickSize:    uint256.MustFromBig(m.TickSize),
			MaxPrice:    uint256.MustFromBig(m.MaxPrice),
			Router:      router,
			HasNative:   m.QuoteAsset == weth || m.BaseAsset == weth,
		})
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(market[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(m.QuoteAsset[:]), Swappable: true},
				{Address: hexutil.Encode(m.BaseAsset[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}

	metadata.Offset, metadata.Pending = end, pending
	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}

// aggregate runs n calls in multicall chunks of listBatchSize.
func (u *PoolsListUpdater) aggregate(ctx context.Context, n int, add func(req *ethrpc.Request, i int)) error {
	for start := 0; start < n; start += listBatchSize {
		req := u.ethrpcClient.NewRequest().SetContext(ctx)
		for i := start; i < min(n, start+listBatchSize); i++ {
			add(req, i)
		}
		if _, err := req.Aggregate(); err != nil {
			return err
		}
	}
	return nil
}
