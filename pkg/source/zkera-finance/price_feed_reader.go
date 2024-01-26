package zkerafinance

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
			"liquiditySource": DexType,
			"reader":          "PriceFeedReader",
		}),
	}
}

func (r *PriceFeedReader) Read(ctx context.Context, address string) (*PriceFeed, error) {
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

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("error when call rpcRequest.Aggregate: %s | %s", err.Error(), address)
		return nil, err
	}

	if v0 == nil {
		v0 = bignumber.ZeroBI
	}
	if v1 == nil {
		v1 = bignumber.ZeroBI
	}

	priceFeed.LatestAnswers[maximizeTrue] = v0
	priceFeed.LatestAnswers[maximizeFalse] = v1

	return priceFeed, nil
}
