package ekubo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	config       *Config
	dataFetchers *dataFetchers

	initialized         bool
	supportedExtensions map[common.Address]ExtensionType
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       cfg,
		dataFetchers: NewDataFetchers(ethrpcClient, cfg),

		supportedExtensions: cfg.SupportedExtensions(),
	}
}

type (
	PoolData struct {
		Token0      common.Address `json:"token0"`
		Token1      common.Address `json:"token1"`
		Fee         string         `json:"fee"`
		TickSpacing uint32         `json:"tick_spacing"`
		Extension   common.Address `json:"extension"`
	}

	GetAllPoolsResult = []PoolData
)

func getStaticPoolKeys() ([]*pools.PoolKey, error) {
	var allPools GetAllPoolsResult
	if err := json.Unmarshal(staticPoolKeysJSON, &allPools); err != nil {
		return nil, fmt.Errorf("decode static pool keys: %w", err)
	}

	newPoolKeys := make([]*pools.PoolKey, 0, len(allPools))
	for _, p := range allPools {
		fee, err := strconv.ParseUint(p.Fee[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
		}

		newPoolKeys = append(newPoolKeys, pools.NewPoolKey(
			p.Token0,
			p.Token1,
			pools.PoolConfig{
				Fee:         fee,
				TickSpacing: p.TickSpacing,
				Extension:   p.Extension,
			}))
	}

	return newPoolKeys, nil
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("Start updating pools list...")
	defer func() {
		logger.Infof("Finish updating pools list.")
	}()

	if u.initialized {
		return nil, nil, nil
	}

	poolKeys, err := getStaticPoolKeys()
	if err != nil {
		return nil, nil, err
	}

	ekuboPools, err := u.dataFetchers.fetchPools(ctx, poolKeys, nil)
	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(poolKeys))
	for i, poolKey := range poolKeys {
		staticExtraBytes, err := json.Marshal(StaticExtra{
			Core:          u.config.Core,
			ExtensionType: u.supportedExtensions[poolKey.Config.Extension],
			PoolKey:       poolKey,
		})
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(Extra(ekuboPools[i]))
		if err != nil {
			return nil, nil, err
		}

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   poolAddress,
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.ZeroToWrappedLower(poolKey.Token0.String(), u.config.ChainId),
					Swappable: true,
				},
				{
					Address:   valueobject.ZeroToWrappedLower(poolKey.Token1.String(), u.config.ChainId),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
			BlockNumber: ekuboPools[i].blockNumber,
		})
	}

	u.initialized = true

	return pools, nil, nil
}
