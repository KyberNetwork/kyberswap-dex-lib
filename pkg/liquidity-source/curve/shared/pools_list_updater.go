package shared

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *Config
		client       *resty.Client
		ethrpcClient *ethrpc.Client
		logger       logger.Logger
	}

	Config struct {
		DexID       string              `mapstructure:"dexID" json:"dexID,omitempty"`
		ChainCode   string              `mapstructure:"chain_code" json:"chain_code,omitempty"`
		ChainID     valueobject.ChainID `mapstructure:"chain_id" json:"chain_id,omitempty"`
		HTTPConfig  HTTPConfig          `mapstructure:"http_config" json:"http_config,omitempty"`
		DataSources []CurveDataSource   `mapstructure:"data_sources" json:"data_sources,omitempty"`

		FetchPoolsMinDuration durationjson.Duration `mapstructure:"fetch_pools_min_duration" json:"fetch_pools_min_duration,omitempty"`
	}

	HTTPConfig struct {
		BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
		Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
		RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
	}
)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client, logger logger.Logger) *PoolsListUpdater {
	client := resty.New().
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)

	return &PoolsListUpdater{
		config:       config,
		client:       client,
		ethrpcClient: ethrpcClient,
		logger:       logger,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte, poolTypeSet mapset.Set[CurvePoolType]) ([]CurvePoolWithType, []byte, error) {
	var pools []CurvePoolWithType

	// pool list doesn't get changed often, so only fetch after some minutes
	now := time.Now().UTC()
	var metadata PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			u.logger.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to unmarshal metadataBytes")
			return nil, nil, err
		}

		if now.Sub(metadata.LastRun) < u.config.FetchPoolsMinDuration.Duration {
			u.logger.Debugf("skip fetching new pool %v %v", now, metadata.LastRun)
			return nil, metadataBytes, nil
		}
	}
	metadata.LastRun = now

	typeCount := map[CurvePoolType]int{}

	for _, dataSource := range u.config.DataSources {
		rawPools, err := u.GetNewPoolsFromDataSource(ctx, dataSource)
		if err != nil {
			return nil, nil, err
		}

		typeMap, err := u.ClassifyPools(ctx, dataSource, rawPools)
		if err != nil {
			return nil, nil, err
		}

		for _, rawPool := range rawPools {
			poolType, ok := typeMap[rawPool.Address]
			if !ok {
				u.logger.Debugf("unknown Curve pool type %s", rawPool.Address)
				continue
			}
			typeCount[poolType] += 1

			if !poolTypeSet.Contains(poolType) {
				u.logger.Debugf("ignore Curve pool type %s %s", poolType, rawPool.Address)
				continue
			}

			pools = append(pools, CurvePoolWithType{
				CurvePool: rawPool,
				PoolType:  poolType,
			})
		}
	}
	u.logger.Infof("fetched %d pools, raw type count: %v", len(pools), typeCount)

	return pools, metadata.ToBytes(), nil
}

func (u *PoolsListUpdater) GetNewPoolsFromDataSource(ctx context.Context, dataSource CurveDataSource) ([]CurvePool, error) {
	u.logger.Infof("fetching pool from %s", dataSource)
	req := u.client.R().SetContext(ctx)

	var result GetPoolsResult

	resp, err := req.SetResult(&result).Get(fmt.Sprintf(getPoolsEndpoint, u.config.ChainCode, dataSource))
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() || !result.Success {
		return nil, errors.WithMessagef(ErrGetPoolsFailed, "[curve] response status: %v, response error: %v, result status %v", resp.Status(), resp.Error(), result.Success)
	}

	// normalize
	for i := range result.Data.PoolData {
		result.Data.PoolData[i].Address = strings.ToLower(result.Data.PoolData[i].Address)
		for j := range result.Data.PoolData[i].Coins {
			if strings.EqualFold(result.Data.PoolData[i].Coins[j].Address, valueobject.NativeAddress) {
				result.Data.PoolData[i].Coins[j].Address = strings.ToLower(valueobject.WrappedNativeMap[u.config.ChainID])
				result.Data.PoolData[i].Coins[j].IsOrgNative = true
			}
		}
	}
	u.logger.Infof("fetched %d pool from %s", len(result.Data.PoolData), dataSource)

	return result.Data.PoolData, nil
}
