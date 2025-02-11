package wombatstable

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (p *PoolsListUpdater) InitPool(_ context.Context) error {
	return nil
}

func (p *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId": p.config.DexID,
		"type":  DexType,
	}).Info("start getting new pools")
	defer func(start time.Time) {
		logger.WithFields(logger.Fields{
			"dexId":    p.config.DexID,
			"type":     DexType,
			"duration": time.Since(start).String(),
		}).Infof("finish getting new pools")
	}(time.Now())

	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId": p.config.DexID,
				"type":  DexType,
			}).Error(err.Error())
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
		return nil, metadataBytes, nil
	}

	if len(newPoolAddresses) > p.config.NewPoolLimit {
		newPoolAddresses = newPoolAddresses[:p.config.NewPoolLimit]
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

	return pools, newMetadataBytes, nil
}

func (p *PoolsListUpdater) getNewPoolAddresses(ctx context.Context, metadata Metadata) ([]common.Address, error) {
	var addresses []common.Address

	req := p.ethrpcClient.R()
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: p.config.RegistryAddress,
		Method: registryMethodGetPools,
	}, []interface{}{&addresses})

	if _, err := req.Call(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": p.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
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
			Target: p.config.LensAddress,
			Method: lensMethodQueryPool,
			Params: []interface{}{poolAddress},
		}, []interface{}{&poolData[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": p.config.DexID,
			"type":  DexType,
		}).Error(err.Error())
		return nil, err
	}

	wrappers := make(map[string]string)
	for t, w := range p.config.Wrappers {
		wrappers[strings.ToLower(t)] = strings.ToLower(w)
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
			logger.WithFields(logger.Fields{
				"dexId": p.config.DexID,
				"type":  DexType,
			}).Error(err.Error())
			return nil, err
		}

		staticExtra := StaticExtra{
			Vault:    p.config.VaultAddress,
			Wrappers: wrappers,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId": p.config.DexID,
				"type":  DexType,
			}).Error(err.Error())
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     poolAddr,
			Exchange:    p.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    poolDat.PoolReserves,
			Tokens:      poolDat.Tokens,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}
