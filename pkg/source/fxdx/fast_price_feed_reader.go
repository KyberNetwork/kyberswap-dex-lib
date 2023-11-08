package fxdx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FastPriceFeedReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewFastPriceFeedReader(ethrpcClient *ethrpc.Client) *FastPriceFeedReader {
	return &FastPriceFeedReader{
		abi:          fastPriceFeedABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"dexType": DexTypeFxdx,
			"reader":  "FastPriceFeedReader",
		}),
	}
}

func (r *FastPriceFeedReader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*FastPriceFeed, error) {
	fastPriceFeed := NewFastPriceFeed()

	if err := r.readData(ctx, address, fastPriceFeed); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, err
	}

	if err := r.readTokenData(ctx, address, fastPriceFeed, tokens); err != nil {
		r.log.Errorf("error when read token data: %s", err)
		return nil, err
	}

	return fastPriceFeed, nil
}

func (r *FastPriceFeedReader) readData(ctx context.Context, address string, fastPriceFeed *FastPriceFeed) error {

	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodDisableFastPriceVoteCount, nil), []interface{}{&fastPriceFeed.DisableFastPriceVoteCount})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodIsSpreadEnabled, nil), []interface{}{&fastPriceFeed.IsSpreadEnabled})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodLastUpdatedAt, nil), []interface{}{&fastPriceFeed.LastUpdatedAt})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodMaxDeviationBasisPoints, nil), []interface{}{&fastPriceFeed.MaxDeviationBasisPoints})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodMinAuthorizations, nil), []interface{}{&fastPriceFeed.MinAuthorizations})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodPriceDuration, nil), []interface{}{&fastPriceFeed.PriceDuration})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodMaxPriceUpdateDelay, nil), []interface{}{&fastPriceFeed.MaxPriceUpdateDelay})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodSpreadBasisPointsIfChainError, nil), []interface{}{&fastPriceFeed.SpreadBasisPointsIfChainError})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodSpreadBasisPointsIfInactive, nil), []interface{}{&fastPriceFeed.SpreadBasisPointsIfInactive})

	_, err := rpcRequest.TryAggregate()
	return err

}

func (r *FastPriceFeedReader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeed,
	tokens []string,
) error {
	tokensLen := len(tokens)

	prices := make([]*big.Int, tokensLen)
	maxCumulativeDeltaDiffs := make([]*big.Int, tokensLen)
	priceData := make([][4]*big.Int, tokensLen)
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodPrices, []interface{}{common.HexToAddress(token)}), []interface{}{&prices[i]})
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodMaxCumulativeDeltaDiffs, []interface{}{common.HexToAddress(token)}), []interface{}{&maxCumulativeDeltaDiffs[i]})
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodGetPriceData, []interface{}{common.HexToAddress(token)}), []interface{}{&priceData[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
		return err
	}

	for i, token := range tokens {
		fastPriceFeed.Prices[token] = prices[i]
		fastPriceFeed.MaxCumulativeDeltaDiffs[token] = maxCumulativeDeltaDiffs[i]
		fastPriceFeed.PriceData[token] = PriceDataItem{
			RefPrice:            priceData[i][0],
			RefTime:             priceData[i][1],
			CumulativeRefDelta:  priceData[i][2],
			CumulativeFastDelta: priceData[i][3],
		}
	}

	return nil
}
