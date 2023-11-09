package velocorev2stable

import (
	"context"
	"encoding/json"
	"strings"
	"time"

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
		logger.Infof("no new pool")
		return nil, metadataBytes, nil
	}

	if len(newPoolAddresses) > p.Config.NewPoolLimit {
		newPoolAddresses = newPoolAddresses[:p.Config.NewPoolLimit]
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
		logger.Infof("got %d new pools", len(pools))
	}

	return pools, newMetadataBytes, nil
}

func (p *PoolsListUpdater) getNewPoolAddresses(ctx context.Context, metadata Metadata) ([]common.Address, error) {
	var addresses []common.Address

	req := p.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: p.Config.RegistryAddress,
		Method: registryMethodGetPools,
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
	var (
		pools    = make([]entity.Pool, len(poolAddreses))
		poolData = make([]poolDataResp, len(poolAddreses))
	)

	req := p.ethrpcClient.R()

	for i, poolAddress := range poolAddreses {
		req.AddCall(&ethrpc.Call{
			ABI:    lensABI,
			Target: p.Config.LensAddress,
			Method: lensMethodQueryPool,
			Params: []interface{}{poolAddress},
		}, []interface{}{&poolData[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to get pools, err: %v", err)
		return nil, err
	}

	for i, poolAddress := range poolAddreses {
		poolAddr := strings.ToLower(poolAddress.Hex())
		poolDat := newPoolData(poolData[i])

		extra := Extra{
			Amp:             poolDat.Amp,
			Fee1e18:         poolDat.Fee1e18,
			LpTokenBalances: poolDat.LpTokenBalances,
			TokenInfo:       nil,
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.Errorf("failed to marshal extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:   poolAddr,
			Exchange:  p.Config.DexID,
			Type:      DexTypeVelocoreV2Stable,
			Timestamp: time.Now().Unix(),
			Reserves:  poolDat.PoolReserves,
			Tokens:    poolDat.Tokens,
			Extra:     string(extraBytes),
		}
	}

	return pools, nil
}
