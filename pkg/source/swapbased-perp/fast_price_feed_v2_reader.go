package swapbasedperp

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FastPriceFeedV2Reader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewFastPriceFeedV2Reader(ethrpcClient *ethrpc.Client) *FastPriceFeedV2Reader {
	return &FastPriceFeedV2Reader{
		abi:          fastPriceFeedV2ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeSwapBasedPerp,
			"reader":          "FastPriceFeedV2Reader",
		}),
	}
}

func (r *FastPriceFeedV2Reader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*FastPriceFeedV2, error) {
	fastPriceFeedV2 := NewFastPriceFeedV2()

	if err := r.readData(ctx, address, fastPriceFeedV2); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, err
	}

	if err := r.readTokenData(ctx, address, fastPriceFeedV2, tokens); err != nil {
		r.log.Errorf("error when read token data: %s", err)
		return nil, err
	}

	return fastPriceFeedV2, nil
}

// readData
// - DisableFastPriceVoteCount
// - IsSpreadEnabled
// - LastUpdatedAt
// - MaxDeviationBasisPoints
// - MinAuthorizations
// - PriceDuration
// - MaxPriceUpdateDelay
// - SpreadBasisPointsIfChainError
// - SpreadBasisPointsIfInactive
func (r *FastPriceFeedV2Reader) readData(ctx context.Context, address string, fastPriceFeed *FastPriceFeedV2) error {

	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2DisableFastPriceVoteCount, nil), []interface{}{&fastPriceFeed.DisableFastPriceVoteCount})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2IsSpreadEnabled, nil), []interface{}{&fastPriceFeed.IsSpreadEnabled})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2LastUpdatedAt, nil), []interface{}{&fastPriceFeed.LastUpdatedAt})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2MaxDeviationBasisPoints, nil), []interface{}{&fastPriceFeed.MaxDeviationBasisPoints})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2MinAuthorizations, nil), []interface{}{&fastPriceFeed.MinAuthorizations})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2PriceDuration, nil), []interface{}{&fastPriceFeed.PriceDuration})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2MaxPriceUpdateDelay, nil), []interface{}{&fastPriceFeed.MaxPriceUpdateDelay})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2SpreadBasisPointsIfChainError, nil), []interface{}{&fastPriceFeed.SpreadBasisPointsIfChainError})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2SpreadBasisPointsIfInactive, nil), []interface{}{&fastPriceFeed.SpreadBasisPointsIfInactive})

	_, err := rpcRequest.TryAggregate()
	return err

}

func (r *FastPriceFeedV2Reader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeedV2,
	tokens []string,
) error {
	tokensLen := len(tokens)

	prices := make([]*big.Int, tokensLen)
	maxCumulativeDeltaDiffs := make([]*big.Int, tokensLen)
	priceData := make([]PriceDataItem, tokensLen)
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2Prices, []interface{}{common.HexToAddress(token)}), []interface{}{&prices[i]})
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2MaxCumulativeDeltaDiffs, []interface{}{common.HexToAddress(token)}), []interface{}{&maxCumulativeDeltaDiffs[i]})
		rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV2GetPriceData, []interface{}{common.HexToAddress(token)}), []interface{}{&priceData[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
		return err
	}

	for i, token := range tokens {
		fastPriceFeed.Prices[token] = prices[i]
		fastPriceFeed.MaxCumulativeDeltaDiffs[token] = maxCumulativeDeltaDiffs[i]
		fastPriceFeed.PriceData[token] = priceData[i]
	}

	return nil
}
