package kyberpmm

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Kyber PMM] Start getting new states for pool %v", p.Address)

	if len(p.Tokens) != 2 {
		err := errors.New("number of tokens should be 2")
		logger.Errorf(err.Error())

		return entity.Pool{}, err
	}

	extra := Extra{}

	priceLevels, inventory, err := t.getPriceLevelsForPool(ctx, p)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to get price levels for pool")
		return entity.Pool{}, err
	}
	//this is supposed to be float
	p.Reserves = make([]string, 2)
	for i, token := range p.Tokens {
		p.Reserves[i] = inventory[token.Address]
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

// getPriceLevelsForPool returns a PriceItem of that pool
// and a map[tokenAddress]Balance for PMM Inventory
func (t *PoolTracker) getPriceLevelsForPool(ctx context.Context, p entity.Pool) (PriceItem, map[string]string, error) {
	result, err := t.client.ListPriceLevels(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get price levels")
		return PriceItem{}, nil, err
	}

	var (
		staticExtra                   StaticExtra
		baseTokenName, quoteTokenName string
	)
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal static extra data")

		return PriceItem{}, nil, err
	}
	for _, token := range p.Tokens {
		if token.Address == staticExtra.BaseTokenAddress {
			baseTokenName = token.Name
		} else if token.Address == staticExtra.QuoteTokenAddress {
			quoteTokenName = token.Name
		}
	}
	priceLevelsForPool, found1 := result.Prices[staticExtra.PairID]

	baseBalance, found2 := result.Balances[baseTokenName]
	quoteBalance, found3 := result.Balances[quoteTokenName]

	if found1 && found2 && found3 {
		return priceLevelsForPool, map[string]string{
			staticExtra.BaseTokenAddress:  baseBalance,
			staticExtra.QuoteTokenAddress: quoteBalance,
		}, nil
	}

	return PriceItem{}, nil, ErrNoPriceLevelsForPool
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

		// Check to prevent division by 0 panic
		if baseToQuoteAskPrice == 0 {
			logger.Debugf("base to quote ask price is 0, skip it")
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
