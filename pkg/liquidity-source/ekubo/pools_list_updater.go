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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolListUpdater struct {
	config       *Config
	httpClient   *resty.Client
	ethrpcClient *ethrpc.Client

	registeredPools map[string]bool
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

		registeredPools: make(map[string]bool),
	}
}

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]*quoting.PoolKey, error) {
	var allPools GetAllPoolsResult
	resp, err := u.httpClient.R().SetContext(ctx).SetResult(&allPools).Get(getPoolKeysEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get pool keys failed status code: %d, message: %v",
			resp.StatusCode(), util.MaxBytesToString(resp.Body(), 256))
	}

	newPoolKeys := make([]*quoting.PoolKey, 0)
	for _, p := range allPools {
		if !strings.EqualFold(p.CoreAddress, u.config.Core) {
			continue
		}

		fee, err := strconv.ParseUint(p.Fee[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
		}

		poolKey := quoting.NewPoolKey(
			common.HexToAddress(p.Token0),
			common.HexToAddress(p.Token1),
			quoting.Config{
				Fee:         fee,
				TickSpacing: p.TickSpacing,
				Extension:   common.HexToAddress(p.Extension),
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

	poolStates, err := fetchPoolStates(ctx, u.ethrpcClient, u.config.DataFetcher, newPoolKeys)
	if err != nil {
		return nil, nil, err
	}

	newPools := make([]entity.Pool, 0, len(newPoolKeys))
	for i, poolKey := range newPoolKeys {
		extension := poolKey.Config.Extension
		var extensionType pool.ExtensionType
		if eth.IsZeroAddress(extension) {
			extensionType = pool.Base
		} else if ext, ok := u.config.Extensions[strings.ToLower(extension.String())]; ok {
			extensionType = ext
		} else {
			logger.WithFields(logger.Fields{
				"poolKey": poolKey,
			}).Warn("skipping pool key with unknown extension")
			continue
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			ExtensionType: extensionType,
			PoolKey:       poolKey,
		})
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(Extra{poolStates[i]})
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
	}

	return newPools, nil, nil
}
