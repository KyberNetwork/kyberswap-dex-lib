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
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
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

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]quoting.PoolKey, error) {
	var allPools GetAllPoolsResult
	resp, err := u.httpClient.R().SetContext(ctx).SetResult(&allPools).Get(getPoolKeysEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, errors.WithMessagef(ErrGetPoolKeysFailed,
			"status code: %d, message: %v", resp.StatusCode(), util.MaxBytesToString(resp.Body(), 256))
	}

	newPoolKeys := make([]quoting.PoolKey, 0)
	for _, pool := range allPools {
		if !strings.EqualFold(pool.CoreAddress, u.config.Core) {
			continue
		}

		fee, err := strconv.ParseUint(pool.Fee[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
		}

		poolKey := quoting.PoolKey{
			Token0: common.HexToAddress(pool.Token0),
			Token1: common.HexToAddress(pool.Token1),
			Config: quoting.Config{
				Fee:         fee,
				TickSpacing: pool.TickSpacing,
				Extension:   common.HexToAddress(pool.Extension),
			},
		}

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

	newPools := make([]entity.Pool, 0, len(newPoolKeys))
	for _, poolKey := range newPoolKeys {
		extension := poolKey.Config.Extension
		var extensionType ExtensionType
		if eth.IsZeroAddress(extension) {
			extensionType = Base
		} else if ext, ok := u.config.Extensions[strings.ToLower(extension.String())]; ok {
			extensionType = ext
		} else {
			logger.WithFields(logger.Fields{
				"poolKey":   poolKey,
				"extension": extension,
			}).Debug("skipping pool key with unknown extension")
			continue
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			ExtensionType: extensionType,
			PoolKey:       poolKey,
		})
		if err != nil {
			return nil, nil, err
		}

		pool := entity.Pool{
			Address:   strings.ToLower(poolKey.StringId()),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   FromEkuboAddress(poolKey.Token0.String(), u.config.ChainID),
					Swappable: true,
				},
				{
					Address:   FromEkuboAddress(poolKey.Token1.String(), u.config.ChainID),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
		}

		newPools = append(newPools, pool)
	}

	return newPools, nil, nil
}
