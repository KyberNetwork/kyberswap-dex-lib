package kyberpmm

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolTracker struct {
	config *Config
	client IClient
}

func NewPoolTracker(cfg *Config, client IClient) *PoolTracker {
	return &PoolTracker{
		config: cfg,
		client: client,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[Kyber PMM] Start getting new states for pool %v", p.Address)

	if len(p.Tokens) != 2 {
		err := errors.New("number of tokens should be 2")
		logger.Errorf(err.Error())

		return entity.Pool{}, err
	}

	extra := Extra{}

	priceLevels, err := t.getPriceLevelsForPool(ctx, p)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to get price levels for pool")
		return entity.Pool{}, err
	}

	extra.BaseToQuotePriceLevels, extra.QuoteToBasePriceLevels = transformPriceLevels(priceLevels)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Kyber PMM] Finish getting new states for pool %v", p.Address)

	return p, nil
}

func (t *PoolTracker) getPriceLevelsForPool(ctx context.Context, p entity.Pool) (PriceItem, error) {
	priceLevels, err := t.client.ListPriceLevels(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get price levels")
		return PriceItem{}, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal static extra data")

		return PriceItem{}, err
	}

	if priceLevelsForPool, found := priceLevels[staticExtra.PairID]; found {
		return priceLevelsForPool, nil
	}

	return PriceItem{}, ErrNoPriceLevelsForPool
}

// For computing prices based on a quote token amount
// we invert the order book (bids become asks and vice versa)
// new price = 1 / price
// new amount = price * amount
func transformPriceLevels(priceLevels PriceItem) ([]PriceLevel, []PriceLevel) {
	baseToQuotePriceLevels := make([]PriceLevel, 0, len(priceLevels.Bids))
	quoteToBasePriceLevels := make([]PriceLevel, 0, len(priceLevels.Asks))

	for _, bid := range priceLevels.Bids {
		baseToQuoteBidPrice, err := strconv.ParseFloat(bid[0], 64)
		if err != nil {
			continue
		}

		baseToQuoteBidAmountFloat64, err := strconv.ParseFloat(bid[1], 64)
		if err != nil {
			continue
		}

		baseToQuotePriceLevels = append(
			baseToQuotePriceLevels,
			PriceLevel{
				Price:  baseToQuoteBidPrice,
				Amount: baseToQuoteBidAmountFloat64,
			},
		)
	}

	for _, ask := range priceLevels.Asks {
		baseToQuoteAskPrice, err := strconv.ParseFloat(ask[0], 64)
		if err != nil {
			continue
		}
		quoteToBaseBidPrice := 1 / baseToQuoteAskPrice

		baseToQuoteAskAmount, err := strconv.ParseFloat(ask[1], 64)
		if err != nil {
			continue
		}
		quoteToBaseBidAmount := baseToQuoteAskPrice * baseToQuoteAskAmount

		quoteToBasePriceLevels = append(
			quoteToBasePriceLevels,
			PriceLevel{
				Price:  quoteToBaseBidPrice,
				Amount: quoteToBaseBidAmount,
			},
		)
	}

	return baseToQuotePriceLevels, quoteToBasePriceLevels
}
