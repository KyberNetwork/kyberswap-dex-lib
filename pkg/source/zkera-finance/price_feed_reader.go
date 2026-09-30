package zkerafinance

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	var v0, v1 *uint256.Int

	rpcRequest := newRequest(r.ethrpcClient, ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: priceFeedMethodLatestAnswer,
		Params: []any{true},
	}, []any{&v0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: priceFeedMethodLatestAnswer,
		Params: []any{false},
	}, []any{&v1})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("error when call rpcRequest.Aggregate: %s | %s", err.Error(), address)
		return nil, err
	}

	v0BI := u256ToBig(v0)
	if v0BI == nil {
		v0BI = bignumber.ZeroBI
	}
	v1BI := u256ToBig(v1)
	if v1BI == nil {
		v1BI = bignumber.ZeroBI
	}

	priceFeed.LatestAnswers[maximizeTrue] = v0BI
	priceFeed.LatestAnswers[maximizeFalse] = v1BI

	return priceFeed, nil
}
