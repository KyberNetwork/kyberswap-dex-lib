package aximabin

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
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	}).Info("start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("finish updating state.")
	}()

	var extra Extra

	poolData, err := t.fetchPoolData(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Errorf("failed to fetch pair data: %v", err)

		// In case of fetching pool state error, we will update pool.Extra.QuoteAvailable = false,
		// so the pool will not be used for routing.
		extra.QuoteAvailable = false

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return entity.Pool{}, err
		}
		p.Extra = string(extraBytes)

		return p, nil
	}

	reserves := []string{poolData.TotalToken0Available, poolData.TotalToken1Available}

	extra.QuoteAvailable = poolData.QuoteAvailable
	extra.MaxAge = t.config.MaxAge

	if bids, err := convertAximaBins(poolData.Depth.Bids, true); err != nil {
		return entity.Pool{}, err
	} else {
		extra.Bids = bids
	}

	if asks, err := convertAximaBins(poolData.Depth.Asks, false); err != nil {
		return entity.Pool{}, err
	} else {
		extra.Asks = asks
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

func (t *PoolTracker) fetchPoolData(ctx context.Context, poolAddress string) (PoolData, error) {
	var poolData PoolData
	_, err := t.client.R().
		SetContext(ctx).
		SetResult(&poolData).
		Get(fmt.Sprintf("/%s/%s/bid_ask", t.config.ChainID.String(), poolAddress))

	if err != nil {
		return PoolData{}, err
	}

	return poolData, nil
}

func convertAximaBins(aximaBins []AximaBin, isBid bool) ([]Bin, error) {
	bins := make([]Bin, len(aximaBins))
	for i, bin := range aximaBins {
		priceF, err := strconv.ParseFloat(bin.Price, 64)
		if err != nil {
			return nil, err
		}

		rate := lo.Ternary(isBid, priceF/Q64, Q64/priceF)

		pie6, _ := strconv.ParseInt(bin.PriceImpactE6, 10, 64)
		bins[i] = Bin{
			BinIdx:           bin.BinIdx,
			Rate:             rate,
			CumulativeVolume: bignumber.NewBig(bin.Price),
			PriceImpactE6:    int(pie6),
		}
	}

	return bins, nil
}
