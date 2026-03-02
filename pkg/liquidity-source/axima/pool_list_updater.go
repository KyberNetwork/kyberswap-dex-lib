package axima

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config *Config
	client *resty.Client
}

var _ = poollist.RegisterFactoryC(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config) *PoolsListUpdater {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)

	return &PoolsListUpdater{config: config, client: client}
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

	pairMetadata, err := u.fetchPairMetadata(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Errorf("failed to fetch pair metadata: %v", err)
		return nil, metadataBytes, err
	}

	pools := lo.Map(pairMetadata, func(pm PairMetadata, _ int) entity.Pool {
		staticExtra := StaticExtra{Pair: pm.Pair}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return entity.Pool{}
		}

		extra, reserves, err := fetchPoolState(ctx, u.client, u.config,
			lo.Ternary(u.config.IsV2, pm.PoolAddress, pm.Pair))
		if err != nil {
			return entity.Pool{}
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return entity.Pool{}
		}

		return entity.Pool{
			Address:  strings.ToLower(pm.PoolAddress),
			Reserves: reserves,
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(pm.Token0), Swappable: true},
				{Address: strings.ToLower(pm.Token1), Swappable: true},
			},
			Exchange:    u.config.DexID,
			Type:        DexType,
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
			Timestamp:   time.Now().Unix(),
		}
	})

	pools = lo.Filter(pools, func(p entity.Pool, _ int) bool {
		return p.Address != ""
	})

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) fetchPairMetadata(ctx context.Context) ([]PairMetadata, error) {
	var pairMetadata []PairMetadata

	_, err := u.client.R().
		SetContext(ctx).
		SetResult(&pairMetadata).
		Get(fmt.Sprintf("/%s/metadata", u.config.ChainID.String()))

	if err != nil {
		return nil, err
	}

	return pairMetadata, nil
}
