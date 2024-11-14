package sfrxeth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Info("Start getting new pools")

	startTime := time.Now()
	u.hasInitialized = true

	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, nil, errors.New("misconfigured poolPath")
	}

	var poolItem PoolItem
	if err := json.Unmarshal(byteData, &poolItem); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, nil, err
	}

	totalSupply, totalAssets, extra, blockNumber, err := getState(
		ctx,
		poolItem.FrxETHMinterAddress,
		poolItem.SfrxETHAddress,
		u.ethrpcClient,
	)

	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:   poolItem.FrxETHMinterAddress,
			Reserves:  []string{totalAssets.String(), totalSupply.String()},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.WrapETHLower(valueobject.EtherAddress, u.config.ChainID),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(poolItem.SfrxETHAddress),
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}
