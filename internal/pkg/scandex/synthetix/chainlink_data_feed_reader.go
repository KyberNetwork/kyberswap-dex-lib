package synthetix

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

const (
	ChainlinkDataFeedMethodLatestRoundData = "latestRoundData"
	ChainlinkDataFeedMethodGetRoundData    = "getRoundData"
)

type ChainlinkDataFeedReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewChainlinkDataFeedReader(scanService *service.ScanService) *ChainlinkDataFeedReader {
	return &ChainlinkDataFeedReader{
		abi:         abis.SynthetixChainlinkDataFeed,
		scanService: scanService,
	}
}

func (r *ChainlinkDataFeedReader) Read(ctx context.Context, address string, roundCount int) (*ChainlinkDataFeed, error) {
	chainlinkDataFeed := NewChainlinkDataFeed()

	if err := r.getLatestRoundData(ctx, address, chainlinkDataFeed); err != nil {
		return nil, err
	}

	if err := r.getHistoryRoundData(ctx, address, chainlinkDataFeed, roundCount); err != nil {
		return nil, err
	}

	return chainlinkDataFeed, nil
}

func (r *ChainlinkDataFeedReader) getLatestRoundData(ctx context.Context, address string, chainlinkDataFeed *ChainlinkDataFeed) error {
	var latestRoundData RoundData

	err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: ChainlinkDataFeedMethodLatestRoundData,
		Output: &latestRoundData,
	})
	if err != nil {
		logger.Errorf("address: %s", address)
		return err
	}

	chainlinkDataFeed.RoundID = latestRoundData.RoundId
	chainlinkDataFeed.Answer = latestRoundData.Answer
	chainlinkDataFeed.StartedAt = latestRoundData.StartedAt
	chainlinkDataFeed.UpdatedAt = latestRoundData.UpdatedAt
	chainlinkDataFeed.AnsweredInRound = latestRoundData.AnsweredInRound
	chainlinkDataFeed.Answers[latestRoundData.RoundId.String()] = latestRoundData

	return nil
}

func (r *ChainlinkDataFeedReader) getHistoryRoundData(ctx context.Context, address string, chainlinkDataFeed *ChainlinkDataFeed, roundCount int) error {
	if roundCount < 2 {
		return nil
	}

	roundDataList := make([]RoundData, roundCount-1)
	var calls []*repository.CallParams
	for i := 1; i < roundCount; i++ {
		roundID := new(big.Int).Sub(chainlinkDataFeed.RoundID, big.NewInt(int64(i)))

		calls = append(calls, &repository.CallParams{
			ABI:    r.abi,
			Target: address,
			Method: ChainlinkDataFeedMethodGetRoundData,
			Params: []interface{}{roundID},
			Output: &roundDataList[i-1],
		})
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	for _, roundData := range roundDataList {
		chainlinkDataFeed.Answers[roundData.RoundId.String()] = roundData
	}

	return nil
}
