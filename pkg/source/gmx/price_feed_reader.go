package gmx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PriceFeedType int

const (
	PriceFeedTypeLatestRoundData PriceFeedType = iota
	PriceFeedTypeLatestRoundAnswer
	PriceFeedTypeDirect // should prefer direct if VaultPriceFeed exposes getPrimaryPrice method
)

type PriceFeedReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger

	PriceFeedType PriceFeedType
}

func NewPriceFeedReader(ethrpcClient *ethrpc.Client) *PriceFeedReader {
	return NewPriceFeedReaderWithParam(ethrpcClient, PriceFeedTypeLatestRoundData)
}

func NewPriceFeedReaderWithParam(ethrpcClient *ethrpc.Client, priceFeedType PriceFeedType) *PriceFeedReader {
	return &PriceFeedReader{
		abi:          priceFeedABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmx,
			"reader":          "PriceFeedReader",
		}),

		PriceFeedType: priceFeedType,
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

	switch r.PriceFeedType {
	case PriceFeedTypeLatestRoundData:
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
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

	case PriceFeedTypeLatestRoundAnswer:
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: "latestRound",
			Params: nil,
		}, []interface{}{&latestRound})
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
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

	case PriceFeedTypeDirect: // already read directly by VaultPriceFeedReader
		return nil
	}

	return nil
}

func (r *PriceFeedReader) getHistoryRoundData(ctx context.Context, address string, priceFeed *PriceFeed,
	roundCount int) error {
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
