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

type FastPriceFeedV1Reader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewFastPriceFeedV1Reader(scanService *service.ScanService) *FastPriceFeedV1Reader {
	return &FastPriceFeedV1Reader{
		abi:         abis.MetavaultFastPriceFeedV1,
		scanService: scanService,
	}
}

func (r *FastPriceFeedV1Reader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*FastPriceFeedV1, error) {
	fastPriceFeedV1 := NewFastPriceFeedV1()

	if err := r.readData(ctx, address, fastPriceFeedV1); err != nil {
		return nil, err
	}

	if err := r.readTokenData(ctx, address, fastPriceFeedV1, tokens); err != nil {
		return nil, err
	}

	return fastPriceFeedV1, nil
}

// readData
// - DisableFastPriceVoteCount
// - IsSpreadEnabled
// - LastUpdatedAt
// - MaxDeviationBasisPoints
// - MinAuthorizations
// - PriceDuration
// - VolBasisPoints
func (r *FastPriceFeedV1Reader) readData(ctx context.Context, address string, fastPriceFeed *FastPriceFeedV1) error {
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	calls := []*repository.CallParams{
		callParamsFactory(FastPriceFeedMethodV1DisableFastPriceVoteCount, &fastPriceFeed.DisableFastPriceVoteCount, nil),
		callParamsFactory(FastPriceFeedMethodV1IsSpreadEnabled, &fastPriceFeed.IsSpreadEnabled, nil),
		callParamsFactory(FastPriceFeedMethodV1LastUpdatedAt, &fastPriceFeed.LastUpdatedAt, nil),
		callParamsFactory(FastPriceFeedMethodV1MaxDeviationBasisPoints, &fastPriceFeed.MaxDeviationBasisPoints, nil),
		callParamsFactory(FastPriceFeedMethodV1MinAuthorizations, &fastPriceFeed.MinAuthorizations, nil),
		callParamsFactory(FastPriceFeedMethodV1PriceDuration, &fastPriceFeed.PriceDuration, nil),
		callParamsFactory(FastPriceFeedMethodV1VolBasisPoints, &fastPriceFeed.VolBasisPoints, nil),
	}

	return r.scanService.MultiCall(ctx, calls)
}

func (r *FastPriceFeedV1Reader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeedV1,
	tokens []string,
) error {
	tokenLen := len(tokens)

	prices := make([]*big.Int, tokenLen)

	var calls []*repository.CallParams
	for i, token := range tokens {
		calls = append(calls, &repository.CallParams{
			ABI:    r.abi,
			Target: address,
			Method: FastPriceFeedMethodV1Prices,
			Params: []interface{}{common.HexToAddress(token)},
			Output: &prices[i],
		})
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	for i, token := range tokens {
		fastPriceFeed.Prices[token] = prices[i]
	}

	return nil
}
