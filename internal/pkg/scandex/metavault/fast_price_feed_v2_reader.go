package metavault

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type FastPriceFeedV2Reader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewFastPriceFeedV2Reader(scanService *service.ScanService) *FastPriceFeedV2Reader {
	return &FastPriceFeedV2Reader{
		abi:         abis.MetavaultFastPriceFeedV2,
		scanService: scanService,
	}
}

func (r *FastPriceFeedV2Reader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*FastPriceFeedV2, error) {
	fastPriceFeedV2 := NewFastPriceFeedV2()

	if err := r.readData(ctx, address, fastPriceFeedV2); err != nil {
		return nil, err
	}

	if err := r.readTokenData(ctx, address, fastPriceFeedV2, tokens); err != nil {
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
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	calls := []*repository.CallParams{
		callParamsFactory(FastPriceFeedMethodV2DisableFastPriceVoteCount, &fastPriceFeed.DisableFastPriceVoteCount, nil),
		callParamsFactory(FastPriceFeedMethodV2IsSpreadEnabled, &fastPriceFeed.IsSpreadEnabled, nil),
		callParamsFactory(FastPriceFeedMethodV2LastUpdatedAt, &fastPriceFeed.LastUpdatedAt, nil),
		callParamsFactory(FastPriceFeedMethodV2MaxDeviationBasisPoints, &fastPriceFeed.MaxDeviationBasisPoints, nil),
		callParamsFactory(FastPriceFeedMethodV2MinAuthorizations, &fastPriceFeed.MinAuthorizations, nil),
		callParamsFactory(FastPriceFeedMethodV2PriceDuration, &fastPriceFeed.PriceDuration, nil),
		callParamsFactory(FastPriceFeedMethodV2MaxPriceUpdateDelay, &fastPriceFeed.MaxPriceUpdateDelay, nil),
		callParamsFactory(FastPriceFeedMethodV2SpreadBasisPointsIfChainError, &fastPriceFeed.SpreadBasisPointsIfChainError, nil),
		callParamsFactory(FastPriceFeedMethodV2SpreadBasisPointsIfInactive, &fastPriceFeed.SpreadBasisPointsIfInactive, nil),
	}

	return r.scanService.MultiCall(ctx, calls)
}

func (r *FastPriceFeedV2Reader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeedV2,
	tokens []string,
) error {
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	tokenLen := len(tokens)

	prices := make([]*big.Int, tokenLen)
	maxCumulativeDeltaDiffs := make([]*big.Int, tokenLen)
	priceData := make([]PriceDataItem, tokenLen)

	var calls []*repository.CallParams
	for i, token := range tokens {
		tokenCalls := []*repository.CallParams{
			callParamsFactory(FastPriceFeedMethodV2Prices, &prices[i], []interface{}{common.HexToAddress(token)}),
			callParamsFactory(FastPriceFeedMethodV2MaxCumulativeDeltaDiffs, &maxCumulativeDeltaDiffs[i], []interface{}{common.HexToAddress(token)}),
			callParamsFactory(FastPriceFeedMethodV2GetPriceData, &priceData[i], []interface{}{common.HexToAddress(token)}),
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	for i, token := range tokens {
		fastPriceFeed.Prices[token] = prices[i]
		fastPriceFeed.MaxCumulativeDeltaDiffs[token] = maxCumulativeDeltaDiffs[i]
		fastPriceFeed.PriceData[token] = priceData[i]
	}

	return nil
}
