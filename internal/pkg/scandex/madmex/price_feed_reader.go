package madmex

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type PriceFeedReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewPriceFeedReader(scanService *service.ScanService) *PriceFeedReader {
	return &PriceFeedReader{
		abi:         abis.GMXPriceFeed,
		scanService: scanService,
	}
}

func (r *PriceFeedReader) Read(ctx context.Context, address string, roundCount int) (*PriceFeed, error) {
	priceFeed := NewPriceFeed()

	if err := r.getLatestRoundData(ctx, address, priceFeed); err != nil {
		return nil, err
	}

	if err := r.getHistoryRoundData(ctx, address, priceFeed, roundCount); err != nil {
		return nil, err
	}

	return priceFeed, nil
}

func (r *PriceFeedReader) getLatestRoundData(ctx context.Context, address string, priceFeed *PriceFeed) error {
	var latestRoundData RoundData

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: PriceFeedMethodLatestRoundData,
		Output: &latestRoundData,
	})
	if err != nil {
		logger.Errorf("address: %s", address)
		return err
	}

	priceFeed.RoundID = latestRoundData.RoundId
	priceFeed.Answer = latestRoundData.Answer
	priceFeed.Answers[latestRoundData.RoundId.String()] = latestRoundData.Answer

	return nil
}

func (r *PriceFeedReader) getHistoryRoundData(ctx context.Context, address string, priceFeed *PriceFeed, roundCount int) error {
	if roundCount < 2 {
		return nil
	}

	roundDataList := make([]RoundData, roundCount-1)
	var calls []*repository.CallParams
	for i := 1; i < roundCount; i++ {
		roundID := new(big.Int).Sub(priceFeed.RoundID, big.NewInt(int64(i)))

		calls = append(calls, &repository.CallParams{
			ABI:    r.abi,
			Target: address,
			Method: PriceFeedMethodGetRoundData,
			Params: []interface{}{roundID},
			Output: &roundDataList[i-1],
		})
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	for _, roundData := range roundDataList {
		priceFeed.Answers[roundData.RoundId.String()] = roundData.Answer
	}

	return nil
}
