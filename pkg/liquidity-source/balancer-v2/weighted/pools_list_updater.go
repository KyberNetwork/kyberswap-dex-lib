package weighted

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var ErrInvalidWeight = errors.New("invalid weight")

type PoolsListUpdater struct {
	config        Config
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(config *Config, _ *ethrpc.Client) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:        config.DexID,
		SubgraphAPI:  config.SubgraphAPI,
		NewPoolLimit: config.NewPoolLimit,
		PoolTypes:    []string{poolTypeWeighted},
	})

	return &PoolsListUpdater{
		config:        *config,
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

	pools, err := u.initPools(ctx, subgraphPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, len(subgraphPools))

	for i, subgraphPool := range subgraphPools {
		pool, err := u.initPool(ctx, subgraphPool)
		if err != nil {
			return nil, err
		}

		pools[i] = pool
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPool(ctx context.Context, subgraphPool *shared.SubgraphPool) (entity.Pool, error) {
	var (
		poolTokens        = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves          = make([]string, len(subgraphPool.Tokens))
		scalingFactors    = make([]*uint256.Int, len(subgraphPool.Tokens))
		normalizedWeights = make([]*uint256.Int, len(subgraphPool.Tokens))

		err error
	)

	for j, token := range subgraphPool.Tokens {
		w, ok := new(big.Float).SetString(token.Weight)
		if !ok {
			return entity.Pool{}, ErrInvalidWeight
		}
		weightStr := new(big.Float).Mul(w, bignumber.BoneFloat).String()
		normalizedWeights[j], err = uint256.FromDecimal(weightStr)
		if err != nil {
			return entity.Pool{}, err
		}

		poolTokens[j] = &entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Swappable: true,
		}

		reserves[j] = "0"

		scalingFactors[j] = number.TenPow(18 - uint8(token.Decimals))
		if subgraphPool.PoolTypeVersion.Int64() > poolTypeVer1 {
			scalingFactors[j] = new(uint256.Int).Mul(scalingFactors[j], number.Number_1e18)
		}
	}

	staticExtra := StaticExtra{
		PoolID:            subgraphPool.ID,
		PoolType:          subgraphPool.PoolType,
		PoolTypeVer:       int(subgraphPool.PoolTypeVersion.Int64()),
		ScalingFactors:    scalingFactors,
		NormalizedWeights: normalizedWeights,
		VaultAddress:      u.config.VaultAddress,
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
