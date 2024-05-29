package dsp

import (
	"context"
	"encoding/json"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
)

type PoolsListUpdater struct {
	config        shared.Config
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(config *shared.Config) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(config)

	return &PoolsListUpdater{
		config:        *config,
		sharedUpdater: sharedUpdater,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := shared.Metadata{
		LastCreatedAtTimestamp: integer.Zero(),
	}
	if len(metadataBytes) > 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			logger.WithFields(logger.Fields{
				"metadataBytes": metadataBytes,
				"error":         err,
			}).Errorf("failed to marshal metadata")

			return nil, metadataBytes, err
		}
	}

	newPools, newMetadata, err := u.sharedUpdater.GetNewPoolsByType(
		ctx,
		PoolType,
		shared.SubgraphPoolTypeDodoStable,
		metadata,
	)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		logger.WithFields(logger.Fields{
			"metadata": metadata,
			"error":    err,
		}).Errorf("failed to marshal metadata")
		return nil, metadataBytes, err
	}

	logger.Infof("got total %v %v pools", len(newPools), PoolType)

	return newPools, newMetadataBytes, nil
}
