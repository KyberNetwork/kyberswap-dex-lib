package gyro3clp

import (
	"context"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"

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
)

type PoolsListUpdater struct {
	config        Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
	graphqlClient *graphqlpkg.Client
}

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

	root3Alphas, err := u.getRoot3Alphas(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, subgraphPools, vaults, root3Alphas)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getRoot3Alphas(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]*big.Int, error) {
	values := make([]*big.Int, len(subgraphPools))

	req := u.ethrpcClient.R()
	for idx, subgraphPool := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: poolMethodGetRoot3Alpha,
		}, []interface{}{&values[idx]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())
		return nil, err
	}

	return values, nil
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
	root3Alphas []*big.Int,
) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))

	for idx := range subgraphPools {
		pool, err := u.initPool(ctx, subgraphPools[idx], vaults[idx], root3Alphas[idx])
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
	root3Alpha *big.Int,
) (entity.Pool, error) {
	var (
		poolTokens      = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves        = make([]string, len(subgraphPool.Tokens))
		scalingFactors  = make([]*uint256.Int, len(subgraphPool.Tokens))
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

	if subgraphPool.PoolTypeVersion != nil {
		poolTypeVersion = int(subgraphPool.PoolTypeVersion.Int64())
	}

	root3AlphaU256, _ := uint256.FromBig(root3Alpha)

	staticExtra := StaticExtra{
		PoolID:         subgraphPool.ID,
		PoolType:       subgraphPool.PoolType,
		PoolTypeVer:    poolTypeVersion,
		ScalingFactors: scalingFactors,
		Root3Alpha:     root3AlphaU256,
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
