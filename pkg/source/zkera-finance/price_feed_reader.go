package zkerafinance

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type PriceFeedReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewPriceFeedReader(ethrpcClient *ethrpc.Client) *PriceFeedReader {
	return &PriceFeedReader{
		abi:          priceFeedABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeZkEra,
			"reader":          "PriceFeedReader",
		}),
	}
}

func (r *PriceFeedReader) Read(ctx context.Context, address string, roundCount int) (*PriceFeed, error) {
	priceFeed := NewPriceFeed()

	var v0, v1 *big.Int

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: priceFeedMethodLatestAnswer,
		Params: []interface{}{true},
	}, []interface{}{&v0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: priceFeedMethodLatestAnswer,
		Params: []interface{}{false},
	}, []interface{}{&v1})

	if _, err := rpcRequest.Aggregate(); err != nil {
		return nil, err
	}

	priceFeed.LatestAnswers[true] = v0
	priceFeed.LatestAnswers[false] = v1

	return priceFeed, nil
}
