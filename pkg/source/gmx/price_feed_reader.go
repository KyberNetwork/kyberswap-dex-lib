package gmx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type Param struct {
	UseLegacyMethod bool
	ABI             abi.ABI
}

type PriceFeedReader struct {
	param        Param
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewPriceFeedReader(ethrpcClient *ethrpc.Client) *PriceFeedReader {
	return NewPriceFeedReaderWithParam(ethrpcClient, Param{
		UseLegacyMethod: true,
		ABI:             PriceFeedABI,
	})
}

func NewPriceFeedReaderWithParam(ethrpcClient *ethrpc.Client, param Param) *PriceFeedReader {
	return &PriceFeedReader{
		param:        param,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmx,
			"reader":          "PriceFeedReader",
		}),
	}
}

func (r *PriceFeedReader) Read(ctx context.Context, address string, roundCount int) (*PriceFeed, error) {
	priceFeed := NewPriceFeed()

	if err := r.getLatestRoundData(ctx, address, priceFeed); err != nil {
		r.log.Errorf("error when get latest round data: %s", err)
		return nil, err
	}

	if err := r.getHistoryRoundData(ctx, address, priceFeed, roundCount); err != nil {
		r.log.Errorf("error when get history round data: %s", err)
		return nil, err
	}

	return priceFeed, nil
}

func (r *PriceFeedReader) getLatestRoundData(ctx context.Context, address string, priceFeed *PriceFeed) error {
	var (
		latestRoundData RoundData
		latestRound     = bignumber.ZeroBI
		latestAnswer    = bignumber.ZeroBI
	)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	if r.param.UseLegacyMethod {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.param.ABI,
			Target: address,
			Method: priceFeedMethodLatestRoundData,
			Params: nil,
		}, []interface{}{&latestRoundData})

		if _, err := rpcRequest.Call(); err != nil {
			return err
		}

		priceFeed.RoundID = latestRoundData.RoundId
		priceFeed.Answer = latestRoundData.Answer
		priceFeed.Answers[latestRoundData.RoundId.String()] = latestRoundData.Answer
	} else {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.param.ABI,
			Target: address,
			Method: "latestRound",
			Params: nil,
		}, []interface{}{&latestRound})
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.param.ABI,
			Target: address,
			Method: "latestAnswer",
			Params: nil,
		}, []interface{}{&latestAnswer})

		if _, err := rpcRequest.Aggregate(); err != nil {
			return err
		}

		priceFeed.RoundID = latestRound
		priceFeed.Answer = latestAnswer
		priceFeed.Answers[latestRound.String()] = latestAnswer
	}

	return nil
}

func (r *PriceFeedReader) getHistoryRoundData(ctx context.Context, address string, priceFeed *PriceFeed, roundCount int) error {
	if roundCount < minRoundCount {
		return nil
	}

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	roundDataList := make([]RoundData, roundCount-1)
	for i := 1; i < roundCount; i++ {
		roundID := new(big.Int).Sub(priceFeed.RoundID, big.NewInt(int64(i)))

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.param.ABI,
			Target: address,
			Method: priceFeedMethodGetRoundData,
			Params: []interface{}{roundID},
		}, []interface{}{&roundDataList[i-1]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	for _, roundData := range roundDataList {
		priceFeed.Answers[roundData.RoundId.String()] = roundData.Answer
	}

	return nil
}
