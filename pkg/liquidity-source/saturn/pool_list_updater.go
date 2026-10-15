package saturn

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	u.hasInitialized = true

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, nil)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Executor: executorAddress,
		Asset:    USDCAddress,
		USDat:    USDatAddress,
		Gas:      defaultGas,
	})
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{
		{
			Address:   SwapFacilityAddress,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{defaultReserves, defaultReserves},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(USDCAddress), Symbol: "USDC", Decimals: 6, Swappable: true},
				{Address: strings.ToLower(USDatAddress), Symbol: "USDAT", Decimals: 6, Swappable: true},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
			BlockNumber: blockNumber,
		},
	}, nil, nil
}
