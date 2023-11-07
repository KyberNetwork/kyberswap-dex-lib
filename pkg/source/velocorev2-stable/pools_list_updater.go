package velocorev2stable

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	Config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		Config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (p *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
}

func (p *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	newPoolAddresses, err := p.getNewPoolAddresses(ctx, metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(newPoolAddresses) == 0 {
		logger.Infof("[VelocoreV2 Stable] no new pool")
		return nil, metadataBytes, nil
	}

	pools, err := p.getPools(ctx, newPoolAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadata := Metadata{
		Offset: metadata.Offset + len(newPoolAddresses),
	}
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.Infof("[VelocoreV2 Stable] got %d new pools", len(pools))
	}

	return pools, newMetadataBytes, nil
}

func (p *PoolsListUpdater) getNewPoolAddresses(ctx context.Context, metadata Metadata) ([]common.Address, error) {
	var addresses []common.Address

	req := p.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: p.Config.RegistryAddress,
		Method: p.Config.RegistryAddress,
	}, []interface{}{&addresses})

	if _, err := req.Call(); err != nil {
		logger.Errorf("failed to get pool addresses from registry, err: %v", err)
		return nil, err
	}

	var newAddresses []common.Address
	for i := metadata.Offset; i < len(addresses); i++ {
		newAddresses = append(newAddresses, addresses[i])
	}

	return newAddresses, nil
}

func (p *PoolsListUpdater) getPools(ctx context.Context, poolAddreses []common.Address) ([]entity.Pool, error) {
	var pools []entity.Pool

	// for _, poolAddress := range poolAddreses {
	// 	pool, err := p.getPool(ctx, poolAddress)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	pools = append(pools, pool)
	// }

	return pools, nil
}
