package doppleramm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata is the next block to scan; MarketCreated is append-only.
type Metadata struct {
	LastScannedBlock uint64 `json:"lastScannedBlock"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if err := u.config.withDefaults(); err != nil {
		return nil, metadataBytes, err
	}
	var meta Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, metadataBytes, err
		}
	}
	if meta.LastScannedBlock == 0 {
		meta.LastScannedBlock = u.config.StartBlock
	}

	head, err := u.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}
	if meta.LastScannedBlock > head {
		return nil, metadataBytes, nil
	}
	toBlock := min(meta.LastScannedBlock+u.config.MaxBlockRange-1, head)

	logs, err := u.ethrpcClient.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(meta.LastScannedBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{common.HexToAddress(u.config.AMMAddress)},
		Topics:    [][]common.Hash{{ammABI.Events["MarketCreated"].ID}},
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(logs))
	for _, log := range logs {
		pool, ok, err := u.decodeMarketCreated(log)
		if err != nil {
			return nil, metadataBytes, err
		}
		if ok {
			pools = append(pools, *pool)
		}
	}

	newMetadata, err := json.Marshal(Metadata{LastScannedBlock: toBlock + 1})
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadata, nil
}

type marketCreatedEvent struct {
	MarketID            *big.Int       `abi:"marketId"`
	MarketConfigPointer common.Address `abi:"marketConfigPointer"`
	TokenX              common.Address `abi:"tokenX"`
	TokenY              common.Address `abi:"tokenY"`
	Liquidity           *big.Int       `abi:"liquidity"`
	UniqueKey           *big.Int       `abi:"uniqueKey"`
	LPTokenID           *big.Int       `abi:"lpTokenId"`
	BootstrapX          *big.Int       `abi:"bootstrapX"`
	BootstrapY          *big.Int       `abi:"bootstrapY"`
}

// decodeMarketCreated keeps only markets this source can quote: the spot CPMM
// module and no extension. Any other composition may need component payloads
// that cannot be derived from stored state, so it is rejected rather than
// quoted with guessed bytes.
func (u *PoolsListUpdater) decodeMarketCreated(log types.Log) (*entity.Pool, bool, error) {
	if len(log.Topics) < 4 {
		return nil, false, nil
	}
	module := common.BytesToAddress(log.Topics[1].Bytes())
	extension := common.BytesToAddress(log.Topics[2].Bytes())
	if module != common.HexToAddress(u.config.SpotModuleAddress) || extension != (common.Address{}) {
		return nil, false, nil
	}

	var event marketCreatedEvent
	if err := ammABI.UnpackIntoInterface(&event, "MarketCreated", log.Data); err != nil {
		return nil, false, err
	}
	if event.MarketID == nil || event.TokenX == (common.Address{}) || event.TokenY == (common.Address{}) {
		return nil, false, ErrUnsupportedMarket
	}

	staticExtra, err := json.Marshal(StaticExtra{
		MarketID: hexutil.Encode(common.BigToHash(event.MarketID).Bytes()),
		TokenX:   hexutil.Encode(event.TokenX[:]),
		TokenY:   hexutil.Encode(event.TokenY[:]),
		AMM:      u.config.AMMAddress,
		Quoter:   u.config.QuoterAddress,
	})
	if err != nil {
		return nil, false, err
	}

	// The tracker fills the fees; seed valid JSON so the simulator can load the pool before then.
	extra, err := json.Marshal(Extra{TokenX: hexutil.Encode(event.TokenX[:])})
	if err != nil {
		return nil, false, err
	}

	return &entity.Pool{
		Address:   hexutil.Encode(common.BigToHash(event.MarketID).Bytes()),
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(event.TokenX[:]), Swappable: true},
			{Address: hexutil.Encode(event.TokenY[:]), Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
		BlockNumber: log.BlockNumber,
	}, true, nil
}
