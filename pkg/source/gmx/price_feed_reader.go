package gmx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type PriceFeedReader struct {
	chainId      valueobject.ChainID
	dexId        string
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewPriceFeedReader(chainID valueobject.ChainID, dexID string, ethrpcClient *ethrpc.Client) *PriceFeedReader {
	return &PriceFeedReader{
		chainId:      chainID,
		dexId:        dexID,
		abi:          priceFeedABI,
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
	var latestRoundData RoundData

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	method := priceFeedMethodLatestRoundData
	if r.chainId == valueobject.ChainIDMantle && r.dexId == string(valueobject.ExchangeKTX) {
		method = "latestRound"
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: method,
		Params: nil,
	}, []interface{}{&latestRoundData})

	if _, err := rpcRequest.Call(); err != nil {
		return err
	}

	priceFeed.RoundID = latestRoundData.RoundId
	priceFeed.Answer = latestRoundData.Answer
	priceFeed.Answers[latestRoundData.RoundId.String()] = latestRoundData.Answer

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
			ABI:    r.abi,
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
