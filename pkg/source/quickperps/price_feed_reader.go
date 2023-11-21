package quickperps

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
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
			"liquiditySource": DexTypeQuickperps,
			"reader":          "PriceFeedReader",
		}),
	}
}

func (r *PriceFeedReader) Read(ctx context.Context, address string) (*PriceFeed, error) {
	priceFeed := &PriceFeed{}

	if err := r.read(ctx, address, priceFeed); err != nil {
		r.log.Errorf("error when get latest round data: %s", err)
		return nil, err
	}

	return priceFeed, nil
}

func (r *PriceFeedReader) read(ctx context.Context, address string, priceFeed *PriceFeed) error {
	type State struct {
		Value     *big.Int `json:"value"`
		Timestamp uint32   `json:"timestamp"`
	}
	priceFeedState := State{}

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: priceFeedMethodRead,
		Params: nil,
	}, []interface{}{&priceFeedState})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	priceFeed.Price = priceFeedState.Value
	priceFeed.Timestamp = priceFeedState.Timestamp

	return nil
}
