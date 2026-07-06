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
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = poolpkg.RegisterFactory0(DexTypeMetricPropAMM, NewPoolSimulator)

type MetricPropAMMMetadataResponse struct {
	Data []MetricPropAMMPairMetadata `json:"data"`
}

type MetricPropAMMPairMetadata struct {
	Pair               string `json:"pair"`
	PoolAddress        string `json:"poolAddress"`
	Token0             string `json:"token0"`
	Token1             string `json:"token1"`
	Token0Decimals     uint8  `json:"token0Decimals"`
	Token1Decimals     uint8  `json:"token1Decimals"`
	PoolFactoryAddress string `json:"poolFactoryAddress"`
}

type metricPropAMMBidAsk struct {
	Bid                  string `json:"bidAdj"`
	Ask                  string `json:"askAdj"`
	TotalToken0Available string `json:"totalToken0Available"`
	TotalToken1Available string `json:"totalToken1Available"`
	ServerTs             int64  `json:"serverTs"`
	Depth                Depth  `json:"depth"`
}

func parseMetricPropAMMMetadata(body []byte) ([]MetricPropAMMPairMetadata, error) {
	var resp MetricPropAMMMetadataResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func metricPropAMMPoolFromMetadata(pm MetricPropAMMPairMetadata, cfg *Config) (entity.Pool, error) {
	staticExtraBytes, err := json.Marshal(StaticExtra{Pair: pm.Pair})
	if err != nil {
		return entity.Pool{}, err
	}
	extraBytes, err := json.Marshal(Extra{QuoteAvailable: false, MaxAge: cfg.MaxAge, IsV2: true})
	if err != nil {
		return entity.Pool{}, err
	}
	return entity.Pool{
		Address:  strings.ToLower(pm.PoolAddress),
		Reserves: []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(pm.Token0), Decimals: pm.Token0Decimals, Swappable: true},
			{Address: strings.ToLower(pm.Token1), Decimals: pm.Token1Decimals, Swappable: true},
		},
		Exchange:    cfg.DexID,
		Type:        DexTypeMetricPropAMM,
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
		Timestamp:   time.Now().Unix(),
	}, nil
}

type MetricPropAMMPoolsListUpdater struct {
	config *Config
	client *resty.Client
}

var _ = poollist.RegisterFactoryC(DexTypeMetricPropAMM, NewMetricPropAMMPoolsListUpdater)

func NewMetricPropAMMPoolsListUpdater(config *Config) *MetricPropAMMPoolsListUpdater {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)
	return &MetricPropAMMPoolsListUpdater{config: config, client: client}
}

func (u *MetricPropAMMPoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	res, err := u.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("/public/v1/evm/%d/metadata", u.config.ChainID))
	if err != nil {
		return nil, metadataBytes, err
	} else if res.IsError() {
		return nil, metadataBytes, fmt.Errorf("metadata API error: %s", res.String())
	}

	pairMetadata, err := parseMetricPropAMMMetadata(res.Body())
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := lo.FilterMap(pairMetadata, func(pm MetricPropAMMPairMetadata, _ int) (entity.Pool, bool) {
		p, err := metricPropAMMPoolFromMetadata(pm, u.config)
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexTypeMetricPropAMM, "pool": pm.PoolAddress}).
				Errorf("failed to build pool: %v", err)
			return entity.Pool{}, false
		}
		return p, true
	})

	return pools, metadataBytes, nil
}

type MetricPropAMMPoolTracker struct {
	config *Config
	client *resty.Client
}

var _ = pooltrack.RegisterFactoryC(DexTypeMetricPropAMM, NewMetricPropAMMPoolTracker)

func NewMetricPropAMMPoolTracker(config *Config) *MetricPropAMMPoolTracker {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)
	if config.HTTPConfig.APIKey != "" {
		client = client.SetAuthToken(config.HTTPConfig.APIKey)
	}
	return &MetricPropAMMPoolTracker{config: config, client: client}
}

func (t *MetricPropAMMPoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p)
}

func (t *MetricPropAMMPoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context, p entity.Pool, _ poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p)
}

func (t *MetricPropAMMPoolTracker) getNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	extra, reserves, err := t.fetchState(ctx, strings.ToLower(p.Address))
	if err != nil {
		extra.QuoteAvailable = false
		if extraBytes, mErr := json.Marshal(extra); mErr == nil {
			p.Extra = string(extraBytes)
		}
		return p, nil
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}
	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func (t *MetricPropAMMPoolTracker) fetchState(ctx context.Context, poolAddr string) (Extra, []string, error) {
	var ba metricPropAMMBidAsk
	res, err := t.client.R().
		SetContext(ctx).
		SetResult(&ba).
		Get(fmt.Sprintf("/public/v1/evm/%d/%s/bid_ask", t.config.ChainID, poolAddr))
	if err != nil {
		return Extra{}, nil, err
	} else if res.IsError() {
		return Extra{}, nil, fmt.Errorf("bid_ask API error: %s", res.String())
	}

	return metricPropAMMExtraFromBidAsk(ba, t.config.MaxAge)
}

func metricPropAMMExtraFromBidAsk(ba metricPropAMMBidAsk, maxAge int64) (Extra, []string, error) {
	bids, err := convertAximaBins(ba.Depth.Bids)
	if err != nil {
		return Extra{}, nil, err
	}
	asks, err := convertAximaBins(ba.Depth.Asks)
	if err != nil {
		return Extra{}, nil, err
	}

	extra := Extra{
		InitBid:        bignumber.NewBig(ba.Bid),
		InitAsk:        bignumber.NewBig(ba.Ask),
		QuoteAvailable: true,
		MaxAge:         maxAge,
		IsV2:           true,
		Bids:           bids,
		Asks:           asks,
	}
	reserves := []string{ba.TotalToken0Available, ba.TotalToken1Available}
	return extra, reserves, nil
}
