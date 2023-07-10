package client

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/hashflow"
)

const (
	AuthorizationHeaderKey = "Authorization"

	listMarketMakersPath = "/taker/v1/marketMakers"
	listPriceLevelsPath  = "/taker/v2/price-levels"
)

var (
	ErrListMarketMakersFailed = errors.New("listMarketMarkers failed")
	ErrListPriceLevelsFailed  = errors.New("listPriceLevels failed")
	ErrInvalidValue           = errors.New("invalid value")
)

type httpClient struct {
	client *resty.Client
	config *HTTPConfig
}

func NewHTTPClient(config *HTTPConfig) *httpClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetTimeout(config.Timeout.Duration).
		SetRetryCount(config.RetryCount)

	return &httpClient{
		client: client,
		config: config,
	}
}

func (c *httpClient) ListMarketMakers(ctx context.Context) ([]string, error) {
	queryParams := listMarketMakersQueryParams{
		Source:    c.config.Source,
		NetworkId: strconv.FormatUint(uint64(c.config.ChainID), 10),
	}

	req := c.client.R().
		SetContext(ctx).
		SetHeader(AuthorizationHeaderKey, c.config.APIKey).
		SetQueryParamsFromValues(queryParams.toUrlValues())

	var result listMarketMakersResult
	resp, err := req.SetResult(&result).Get(listMarketMakersPath)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, ErrListMarketMakersFailed
	}

	return result.MarketMakers, nil
}

func (c *httpClient) ListPriceLevels(ctx context.Context, marketMakers []string) ([]hashflow.Pair, error) {
	queryParams := listPriceLevelsQueryParams{
		Source:        c.config.Source,
		NetworkId:     strconv.FormatUint(uint64(c.config.ChainID), 10),
		MarketMarkers: marketMakers,
	}

	req := c.client.R().
		SetContext(ctx).
		SetHeader(AuthorizationHeaderKey, c.config.APIKey).
		SetQueryParamsFromValues(queryParams.toUrlValues())

	var result listPriceLevelsResult
	resp, err := req.SetResult(&result).Get(listPriceLevelsPath)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() || result.Status != "success" {
		return nil, ErrListPriceLevelsFailed
	}

	return transformPairsByMarketMakers(result.Levels), nil
}

func transformPairsByMarketMakers(pairsByMarketMakers map[string][]listPriceLevelsResultPair) []hashflow.Pair {
	var allHashflowPairs []hashflow.Pair

	for marketMaker, pairs := range pairsByMarketMakers {
		hashflowPairs, err := transformPairs(marketMaker, pairs)
		if err != nil {
			continue
		}

		allHashflowPairs = append(allHashflowPairs, hashflowPairs...)
	}

	return allHashflowPairs
}

func transformPairs(marketMaker string, pairs []listPriceLevelsResultPair) ([]hashflow.Pair, error) {
	hashflowPairByKey := make(map[string]hashflow.Pair, len(pairs))
	for _, pair := range pairs {
		shouldReverse := shouldReverseTokenOrder(pair.Pair.BaseToken, pair.Pair.QuoteToken)

		var (
			token0, token1       string
			decimals0, decimals1 uint8
		)
		if shouldReverse {
			token0, token1 = pair.Pair.QuoteToken, pair.Pair.BaseToken
			decimals0, decimals1 = pair.Pair.QuoteTokenDecimals, pair.Pair.BaseTokenDecimals
		} else {
			token0, token1 = pair.Pair.BaseToken, pair.Pair.QuoteToken
			decimals0, decimals1 = pair.Pair.BaseTokenDecimals, pair.Pair.QuoteTokenDecimals
		}

		pairKey := genPairKey(token0, token1)

		hashflowPair, ok := hashflowPairByKey[pairKey]
		if !ok {
			hashflowPair = hashflow.Pair{
				MarketMaker: marketMaker,
				Tokens:      []string{token0, token1},
				Decimals:    []uint8{decimals0, decimals1},
			}
		}

		priceLevels := make([]hashflow.PriceLevel, 0, len(pair.Levels))
		for _, resultPriceLevel := range pair.Levels {
			priceLevel, err := newHashflowPairPriceLevel(resultPriceLevel)
			if err != nil {
				return nil, errors.Wrapf(err, "marketMakers: [%s]", marketMaker)
			}

			priceLevels = append(priceLevels, priceLevel)
		}

		if shouldReverse {
			hashflowPair.OneToZeroPriceLevels = priceLevels
		} else {
			hashflowPair.ZeroToOnePriceLevels = priceLevels
		}

		hashflowPairByKey[pairKey] = hashflowPair
	}

	hashflowPairs := make([]hashflow.Pair, 0, len(hashflowPairByKey))
	for _, pair := range hashflowPairByKey {
		hashflowPairs = append(hashflowPairs, pair)
	}

	return hashflowPairs, nil
}

func newHashflowPairPriceLevel(resultPriceLevel listPriceLevelResultPriceLevel) (hashflow.PriceLevel, error) {
	price, isValid := new(big.Float).SetString(resultPriceLevel.Price)
	if !isValid {
		return hashflow.PriceLevel{}, errors.Wrapf(ErrInvalidValue, "price: [%s]", resultPriceLevel.Price)
	}

	level, isValid := new(big.Float).SetString(resultPriceLevel.Level)
	if !isValid {
		return hashflow.PriceLevel{}, errors.Wrapf(ErrInvalidValue, "level: [%s]", resultPriceLevel.Level)
	}

	return hashflow.PriceLevel{
		Price: price,
		Level: level,
	}, nil
}

// shouldReverseTokenOrder returns true if baseToken > quoteToken
func shouldReverseTokenOrder(baseToken, quoteToken string) bool {
	return baseToken > quoteToken
}

func genPairKey(token0, token1 string) string {
	return fmt.Sprintf("%s-%s", token0, token1)
}
