package susde

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId":   u.config.DexID,
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	if u.initialized {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	asset, err := u.getAsset(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, metadataBytes, err
	}

	pool := entity.Pool{
		Address:  StakedUSDeV2,
		Exchange: u.config.DexID,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: asset, Swappable: true},
			{Address: StakedUSDeV2, Swappable: true},
		},
	}

	u.initialized = true

	return []entity.Pool{pool}, metadataBytes, nil
}

func (u *PoolsListUpdater) getAsset(ctx context.Context) (string, error) {
	var addr common.Address
	req := u.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    stakedUSDeV2ABI,
		Target: StakedUSDeV2,
		Method: stakedUSDeV2MethodAsset,
	}, []interface{}{&addr})
	if _, err := req.Call(); err != nil {
		return "", err
	}
	return strings.ToLower(addr.Hex()), nil
}
