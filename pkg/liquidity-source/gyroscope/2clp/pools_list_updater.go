package gyro2clp

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
	graphqlClient *graphqlpkg.Client
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
		PoolTypes:       []string{poolType},
	}, graphqlClient)

	return &PoolsListUpdater{
		config:        *config,
		ethrpcClient:  ethrpcClient,
		sharedUpdater: sharedUpdater,
		graphqlClient: graphqlClient,
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

	sqrtParameters, err := u.getSqrtParameters(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, subgraphPools, vaults, sqrtParameters)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getSqrtParameters(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([][2]*big.Int, error) {
	sqrtParameters := make([][2]*big.Int, len(subgraphPools))

	req := u.ethrpcClient.R()
	for idx, subgraphPool := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: poolMethodGetSqrtParameters,
		}, []interface{}{&sqrtParameters[idx]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, err
	}

	return sqrtParameters, nil
}

func (u *PoolsListUpdater) getVaults(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]string, error) {
	vaultAddresses := make([]common.Address, len(subgraphPools))
	vaults := make([]string, len(subgraphPools))

	req := u.ethrpcClient.R()
	for idx, subgraphPool := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: poolMethodGetVault,
		}, []interface{}{&vaultAddresses[idx]})
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

func (u *PoolsListUpdater) initPools(
	ctx context.Context,
	subgraphPools []*shared.SubgraphPool,
	vaults []string,
	sqrtParameters [][2]*big.Int,
) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))

	for idx := range subgraphPools {
		pool, err := u.initPool(ctx, subgraphPools[idx], vaults[idx], sqrtParameters[idx])
		if err != nil {
			return nil, err
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPool(
	ctx context.Context,
	subgraphPool *shared.SubgraphPool,
	vault string,
	sqrtParameters [2]*big.Int,
) (entity.Pool, error) {
	var (
		poolTokens      = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves        = make([]string, len(subgraphPool.Tokens))
		scalingFactors  = make([]*uint256.Int, len(subgraphPool.Tokens))
		sqrtParams      = make([]*uint256.Int, len(sqrtParameters))
		poolTypeVersion int
	)

	for j, token := range subgraphPool.Tokens {
		poolTokens[j] = &entity.PoolToken{
			Address:   token.Address,
			Weight:    defaultWeight,
			Swappable: true,
		}

		reserves[j] = "0"

		scalingFactors[j] = new(uint256.Int).Mul(
			number.TenPow(18-uint8(token.Decimals)),
			number.Number_1e18,
		)
	}

	for j, s := range sqrtParameters {
		sqrtParams[j], _ = uint256.FromBig(s)
	}

	if subgraphPool.PoolTypeVersion != nil {
		poolTypeVersion = int(subgraphPool.PoolTypeVersion.Int64())
	}

	staticExtra := StaticExtra{
		PoolID:         subgraphPool.ID,
		PoolType:       subgraphPool.PoolType,
		PoolTypeVer:    poolTypeVersion,
		ScalingFactors: scalingFactors,
		SqrtParameters: sqrtParams,
		Vault:          vault,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
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
	}, nil
}
