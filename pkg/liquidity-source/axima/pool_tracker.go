package axima

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var (
	ErrPoolCollectionNotFound   = errors.New("pool collection not found")
	ErrCollectionByPoolNotFound = errors.New("collection by pool not found")
	ErrPoolDataNotFound         = errors.New("pool data not found")
)

type PoolTracker struct {
	config *Config
	client *resty.Client
}

var _ = pooltrack.RegisterFactoryC(DexType, NewPoolTracker)

func NewPoolTracker(config *Config) *PoolTracker {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)
	return &PoolTracker{config: config, client: client}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":       t.config.DexID,
		"dexType":     DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	pair := staticExtra.Pair
	pairData, err := t.fetchPairData(ctx, pair)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
			"pair":        pair,
		}).Errorf("failed to fetch pair data: %v", err)
		return entity.Pool{}, err
	}

	reserves := []string{pairData.TotalToken0Available, pairData.TotalToken1Available}

	var extra Extra

	bidF, err := strconv.ParseFloat(pairData.Bid, 64)
	if err != nil {
		return entity.Pool{}, err
	}
	extra.ZeroToOneRate = bidF / Q64

	askF, err := strconv.ParseFloat(pairData.Ask, 64)
	if err != nil {
		return entity.Pool{}, err
	}
	extra.OneToZeroRate = Q64 / askF

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) fetchPairData(ctx context.Context, pair string) (PairData, error) {
	var pairData PairData
	_, err := t.client.R().
		SetContext(ctx).
		SetResult(&pairData).
		Get(fmt.Sprintf("/%s/%s/bid_ask", t.config.ChainID.String(), pair))

	if err != nil {
		return PairData{}, err
	}

	return pairData, nil
}
