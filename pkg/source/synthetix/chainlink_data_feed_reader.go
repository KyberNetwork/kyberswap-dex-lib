package synthetix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ChainlinkDataFeedReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewChainlinkDataFeedReader(cfg *Config, ethrpcClient *ethrpc.Client) *ChainlinkDataFeedReader {
	return &ChainlinkDataFeedReader{
		abi:          chainlinkDataFeed,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *ChainlinkDataFeedReader) Read(ctx context.Context, address string, roundCount int) (*ChainlinkDataFeed, error) {
	chainlinkDataFeed := NewChainlinkDataFeed()

	if err := r.getLatestRoundData(ctx, address, chainlinkDataFeed); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not get latest round data")
		return nil, err
	}

	if err := r.getHistoryRoundData(ctx, address, chainlinkDataFeed, roundCount); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not get history round data")
		return nil, err
	}

	return chainlinkDataFeed, nil
}

func (r *ChainlinkDataFeedReader) getLatestRoundData(ctx context.Context, address string, chainlinkDataFeed *ChainlinkDataFeed) error {
	var latestRoundData RoundData

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: ChainlinkDataFeedMethodLatestRoundData,
			Params: nil,
		}, []interface{}{&latestRoundData})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not get latest round data")
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
	// start to get historical rounds data when current round count is greater than 1
	if roundCount <= 1 {
		return nil
	}

	roundDataList := make([]RoundData, roundCount-1)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 1; i < roundCount; i++ {
		roundID := new(big.Int).Sub(chainlinkDataFeed.RoundID, big.NewInt(int64(i)))

		req.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: ChainlinkDataFeedMethodGetRoundData,
			Params: []interface{}{roundID},
		}, []interface{}{&roundDataList[i-1]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not get history round data")
		return err
	}

	for _, roundData := range roundDataList {
		chainlinkDataFeed.Answers[roundData.RoundId.String()] = roundData
	}

	return nil
}
