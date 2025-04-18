package stable

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
}

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:           config.DexID,
		SubgraphAPI:     config.SubgraphAPI,
		SubgraphHeaders: config.SubgraphHeaders,
		NewPoolLimit:    config.NewPoolLimit,
		PoolTypes:       []string{poolTypeStable, poolTypeMetaStable},
	}, graphqlClient)

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

	vaults, err := u.getVaults(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(subgraphPools, vaults)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getVaults(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]string, error) {
	vaultAddresses := make([]common.Address, len(subgraphPools))
	vaults := make([]string, len(subgraphPools))

	req := u.ethrpcClient.R().SetContext(ctx)
	for idx, subgraphPool := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: poolMethodGetVault,
		}, []any{&vaultAddresses[idx]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, err
	}

	for idx, addr := range vaultAddresses {
		vaults[idx] = strings.ToLower(addr.Hex())
	}

	return vaults, nil
}

func (u *PoolsListUpdater) initPools(subgraphPools []*shared.SubgraphPool,
	vaults []string) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))
	for idx := range subgraphPools {
		pool, err := u.initPool(subgraphPools[idx], vaults[idx])
		if err != nil {
			return nil, err
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPool(subgraphPool *shared.SubgraphPool, vault string) (entity.Pool,
	error) {
	var (
		poolTokens     = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves       = make([]string, len(subgraphPool.Tokens))
		scalingFactors = make([]*uint256.Int, len(subgraphPool.Tokens))
		basePools      = make(map[string][]string, len(subgraphPool.Tokens))
	)

	for j, token := range subgraphPool.Tokens {
		poolTokens[j] = &entity.PoolToken{
			Address:   token.Address,
			Weight:    defaultWeight,
			Swappable: true,
		}
		reserves[j] = "0"
		scalingFactors[j] = new(uint256.Int).Mul(
			bignumber.TenPowInt(18-uint8(token.Decimals)),
			bignumber.BONE,
		)

		if token.Token.Pool.Address != "" {
			basePools[token.Address] = []string{}
		}
	}

	poolSpec, err := _getPoolSpecialization(subgraphPool.ID)
	if err != nil {
		return entity.Pool{}, err
	}

	staticExtra := StaticExtra{
		PoolID:             subgraphPool.ID,
		PoolType:           subgraphPool.PoolType,
		PoolTypeVer:        int(subgraphPool.PoolTypeVersion.Int64()),
		PoolSpecialization: poolSpec,
		Vault:              vault,
		BasePools:          basePools,
		BatchSwapEnabled:   u.config.BatchSwapEnabled,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	extra := Extra{ScalingFactors: scalingFactors}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     subgraphPool.Address,
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Tokens:      poolTokens,
		Reserves:    reserves,
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}, nil
}

func _getPoolSpecialization(poolID string) (uint8, error) {
	// uint256(poolId >> (10 * 8)) & (2**(2 * 8) - 1);

	value := new(big.Int).SetBytes(common.FromHex(poolID))
	value = new(big.Int).And(
		new(big.Int).Rsh(value, 80),
		big.NewInt(65535),
	)

	return uint8(value.Uint64()), nil
}
