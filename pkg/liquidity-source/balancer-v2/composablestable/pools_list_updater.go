package composablestable

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:        config.DexID,
		SubgraphAPI:  config.SubgraphAPI,
		NewPoolLimit: config.NewPoolLimit,
		PoolTypes:    []string{poolTypeComposableStable},
	})

	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		sharedUpdater: sharedUpdater,
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

	subgraphPools, newMetadataBytes, err := u.sharedUpdater.GetNewPools(ctx, metadataBytes)
	if err != nil {
		return nil, nil, err
	}

	bptIndexes, err := u.getBptIndex(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, subgraphPools, bptIndexes)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getBptIndex(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]*big.Int, error) {
	bptIndexes := make([]*big.Int, len(subgraphPools))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, p := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: poolMethodGetBptIndex,
		}, []interface{}{&bptIndexes[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return bptIndexes, nil
}

func (u *PoolsListUpdater) initPools(
	ctx context.Context,
	subgraphPools []*shared.SubgraphPool,
	bptIndexes []*big.Int,
) ([]entity.Pool, error) {
	pools := make([]entity.Pool, len(subgraphPools))
	for i, subgraphPool := range subgraphPools {
		pool, err := u.initPool(ctx, subgraphPool, bptIndexes[i])
		if err != nil {
			return nil, err
		}

		pools[i] = pool
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPool(
	ctx context.Context,
	subgraphPool *shared.SubgraphPool,
	bptIndex *big.Int,
) (entity.Pool, error) {
	var (
		poolTokens     = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves       = make([]string, len(subgraphPool.Tokens))
		scalingFactors = make([]*uint256.Int, len(subgraphPool.Tokens))

		err error
	)

	for j, token := range subgraphPool.Tokens {
		poolTokens[j] = &entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Swappable: true,
		}

		reserves[j] = "0"

		scalingFactors[j] = new(uint256.Int).Mul(
			number.TenPow(18-uint8(token.Decimals)),
			number.Number_1e18,
		)
	}

	staticExtra := StaticExtra{
		PoolID:       subgraphPool.ID,
		PoolType:     subgraphPool.PoolType,
		PoolTypeVer:  int(subgraphPool.PoolTypeVersion.Int64()),
		BptIndex:     int(bptIndex.Int64()),
		VaultAddress: strings.ToLower(u.config.VaultAddress),
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     strings.ToLower(subgraphPool.Address),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Tokens:      poolTokens,
		Reserves:    reserves,
		StaticExtra: string(staticExtraBytes),
	}, nil
}
