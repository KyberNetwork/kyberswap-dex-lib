package mimswap

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dodov2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type (
	Config struct {
		ChainId valueobject.ChainID `json:"chainId"`
	}

	PoolsListUpdater struct {
		config         *Config
		ethrpcClient   *ethrpc.Client
		hasInitialized bool
	}
)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	poolAddresses, ok := mimSwapPoolAddress[u.config.ChainId]
	if !ok {
		logger.Errorf("pool ids not configured for chain id %s", u.config.ChainId)
		return nil, nil, nil
	}

	baseToken := make([]common.Address, len(poolAddresses))
	quoteToken := make([]common.Address, len(poolAddresses))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    dodov2.V2PoolABI,
			Target: poolAddress,
			Method: dodov2.DodoV2MethodGetBaseToken,
		}, []any{&baseToken[i]}).AddCall(&ethrpc.Call{
			ABI:    dodov2.V2PoolABI,
			Target: poolAddress,
			Method: dodov2.DodoV2MethodGetQuoteToken,
		}, []any{&quoteToken[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to aggregate request: %s", err)
		return nil, nil, err
	}

	var pools []entity.Pool
	for i, poolAddress := range poolAddresses {
		staticExtraBytes, err := json.Marshal(dodov2.StaticExtra{
			Type: dodov2.SubgraphPoolTypeDodoStable,
		})
		if err != nil {
			logger.Errorf("failed to marshal static extra: %s", err)
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(poolAddress),
			Reserves:  []string{"0", "0"},
			Exchange:  DexType,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(baseToken[i].String()),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(quoteToken[i].String()),
					Swappable: true,
				},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	u.hasInitialized = true

	return pools, nil, nil
}
