package madmex

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type FastPriceFeedV2Reader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewFastPriceFeedV2Reader(scanService *service.ScanService) *FastPriceFeedV2Reader {
	return &FastPriceFeedV2Reader{
		abi:         abis.GMXFastPriceFeedV2,
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
	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2DisableFastPriceVoteCount,
			Params: nil,
			Output: &fastPriceFeed.DisableFastPriceVoteCount,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2IsSpreadEnabled,
			Params: nil,
			Output: &fastPriceFeed.IsSpreadEnabled,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2LastUpdatedAt,
			Params: nil,
			Output: &fastPriceFeed.LastUpdatedAt,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2MaxDeviationBasisPoints,
			Params: nil,
			Output: &fastPriceFeed.MaxDeviationBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2MinAuthorizations,
			Params: nil,
			Output: &fastPriceFeed.MinAuthorizations,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2PriceDuration,
			Params: nil,
			Output: &fastPriceFeed.PriceDuration,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2MaxPriceUpdateDelay,
			Params: nil,
			Output: &fastPriceFeed.MaxPriceUpdateDelay,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2SpreadBasisPointsIfChainError,
			Params: nil,
			Output: &fastPriceFeed.SpreadBasisPointsIfChainError,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV2SpreadBasisPointsIfInactive,
			Params: nil,
			Output: &fastPriceFeed.SpreadBasisPointsIfInactive,
		},
	}

	return r.scanService.MultiCall(ctx, calls)
}

func (r *FastPriceFeedV2Reader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeedV2,
	tokens []string,
) error {
	tokenLen := len(tokens)

	prices := make([]*big.Int, tokenLen)
	maxCumulativeDeltaDiffs := make([]*big.Int, tokenLen)
	priceData := make([]PriceDataItem, tokenLen)

	var calls []*repository.CallParams
	for i, token := range tokens {
		tokenCalls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: FastPriceFeedMethodV2Prices,
				Params: []interface{}{common.HexToAddress(token)},
				Output: &prices[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: FastPriceFeedMethodV2MaxCumulativeDeltaDiffs,
				Params: []interface{}{common.HexToAddress(token)},
				Output: &maxCumulativeDeltaDiffs[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: FastPriceFeedMethodV2GetPriceData,
				Params: []interface{}{common.HexToAddress(token)},
				Output: &priceData[i],
			},
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
