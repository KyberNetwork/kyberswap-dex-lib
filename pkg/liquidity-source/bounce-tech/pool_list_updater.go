package bouncetech

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type Metadata struct {
	Offset int `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	// Fetch all LT addresses from the BounceTech Factory.
	var allLTs []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: "lts",
		}, []any{&allLTs}).Call(); err != nil {
		return nil, metadataBytes, err
	}

	// Determine the base asset (USDC) from GlobalStorage.
	var baseAsset common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    globalStorageABI,
			Target: u.config.GlobalStorageAddress,
			Method: "baseAsset",
		}, []any{&baseAsset}).Call(); err != nil {
		return nil, metadataBytes, err
	}

	if metadata.Offset >= len(allLTs) {
		return nil, metadataBytes, nil
	}

	batchSize := u.config.NewPoolLimit
	if batchSize <= 0 {
		batchSize = 50
	}
	end := metadata.Offset + batchSize
	if end > len(allLTs) {
		end = len(allLTs)
	}
	batch := allLTs[metadata.Offset:end]

	pools := make([]entity.Pool, 0, len(batch))
	for _, lt := range batch {
		p := u.newPool(lt.Hex(), baseAsset.Hex())
		pools = append(pools, p)
	}

	newMetadata := Metadata{Offset: end}
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dexID":     u.config.DexID,
		"new_pools": len(pools),
		"offset":    end,
		"total":     len(allLTs),
	}).Info("[bounce-tech] finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) newPool(ltAddress, usdcAddress string) entity.Pool {
	staticExtra := StaticExtra{USDC: usdcAddress}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	return entity.Pool{
		Address:   ltAddress,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: usdcAddress, Swappable: true},
			{Address: ltAddress, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
		BlockNumber: 0,
		Extra:       "{}",
	}
}
