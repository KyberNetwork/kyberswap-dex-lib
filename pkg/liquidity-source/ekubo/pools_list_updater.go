package ekubo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/pools"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolListUpdater struct {
	config       *Config
	httpClient   *resty.Client
	ethrpcClient *ethrpc.Client
	dataFetchers *dataFetchers

	registeredPools     map[string]bool
	supportedExtensions map[common.Address]ExtensionType
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	httpClient := resty.New().
		SetBaseURL(cfg.HTTPConfig.BaseURL).
		SetTimeout(cfg.HTTPConfig.Timeout.Duration).
		SetRetryCount(cfg.HTTPConfig.RetryCount)

	return &PoolListUpdater{
		config:       cfg,
		httpClient:   httpClient,
		ethrpcClient: ethrpcClient,
		dataFetchers: NewDataFetchers(ethrpcClient, cfg),

		registeredPools:     make(map[string]bool),
		supportedExtensions: cfg.SupportedExtensions(),
	}
}

const getPoolKeysEndpoint = "/v1/poolKeys"

type (
	PoolData struct {
		CoreAddress common.Address `json:"core_address"`
		Token0      common.Address `json:"token0"`
		Token1      common.Address `json:"token1"`
		Fee         string         `json:"fee"`
		TickSpacing uint32         `json:"tick_spacing"`
		Extension   common.Address `json:"extension"`
	}

	GetAllPoolsResult = []PoolData
)

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]*pools.PoolKey, error) {
	var allPools GetAllPoolsResult
	resp, err := u.httpClient.R().SetContext(ctx).SetResult(&allPools).Get(getPoolKeysEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get pool keys failed status code: %d, message: %v",
			resp.StatusCode(), util.MaxBytesToString(resp.Body(), 256))
	}

	newPoolKeys := make([]*pools.PoolKey, 0)
	for _, p := range allPools {
		if p.CoreAddress.Cmp(u.config.Core) != 0 {
			continue
		}

		if _, ok := u.supportedExtensions[p.Extension]; !ok {
			continue
		}

		fee, err := strconv.ParseUint(p.Fee[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
		}

		poolKey := pools.NewPoolKey(
			p.Token0,
			p.Token1,
			pools.PoolConfig{
				Fee:         fee,
				TickSpacing: p.TickSpacing,
				Extension:   p.Extension,
			})

		if u.registeredPools[poolKey.StringId()] {
			continue
		}

		newPoolKeys = append(newPoolKeys, poolKey)
	}

	return newPoolKeys, nil
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("Start updating pools list...")
	defer func() {
		logger.Infof("Finish updating pools list.")
	}()

	newPoolKeys, err := u.getNewPoolKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	newEkuboPools, err := u.dataFetchers.fetchPools(ctx, newPoolKeys)
	if err != nil {
		return nil, nil, err
	}

	newPools := make([]entity.Pool, 0, len(newPoolKeys))
	for i, poolKey := range newPoolKeys {
		extensionType, ok := u.supportedExtensions[poolKey.Config.Extension]
		if !ok {
			logger.WithFields(logger.Fields{
				"poolKey": poolKey,
			}).Warn("skipping pool key with unknown extension")
			continue
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			Core:          u.config.Core,
			ExtensionType: extensionType,
			PoolKey:       poolKey,
		})
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(Extra(newEkuboPools[i]))
		if err != nil {
			return nil, nil, err
		}

		newPools = append(newPools, entity.Pool{
			Address:   strings.ToLower(poolKey.StringId()),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   FromEkuboAddress(poolKey.Token0.String(), u.config.ChainId),
					Swappable: true,
				},
				{
					Address:   FromEkuboAddress(poolKey.Token1.String(), u.config.ChainId),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
		})

		u.registeredPools[poolKey.StringId()] = true
	}

	return newPools, nil, nil
}
