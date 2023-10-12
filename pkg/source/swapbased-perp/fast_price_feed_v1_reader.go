package swapbasedperp

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FastPriceFeedV1Reader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewFastPriceFeedV1Reader(ethrpcClient *ethrpc.Client) *FastPriceFeedV1Reader {
	return &FastPriceFeedV1Reader{
		abi:          fastPriceFeedV1ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeSwapBasedPerp,
			"reader":          "FastPriceFeedV1Reader",
		}),
	}
}

func (r *FastPriceFeedV1Reader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*FastPriceFeedV1, error) {
	fastPriceFeedV1 := NewFastPriceFeedV1()

	if err := r.readData(ctx, address, fastPriceFeedV1); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, err
	}

	if err := r.readTokenData(ctx, address, fastPriceFeedV1, tokens); err != nil {
		r.log.Errorf("error when read token data: %s", err)
		return nil, err
	}

	return fastPriceFeedV1, nil
}

// readData
// - DisableFastPriceVoteCount
// - IsSpreadEnabled
// - LastUpdatedAt
// - MaxDeviationBasisPoints
// - MinAuthorizations
// - PriceDuration
// - VolBasisPoints
func (r *FastPriceFeedV1Reader) readData(ctx context.Context, address string, fastPriceFeed *FastPriceFeedV1) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1DisableFastPriceVoteCount, nil), []interface{}{&fastPriceFeed.DisableFastPriceVoteCount})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1IsSpreadEnabled, nil), []interface{}{&fastPriceFeed.IsSpreadEnabled})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1LastUpdatedAt, nil), []interface{}{&fastPriceFeed.LastUpdatedAt})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1MaxDeviationBasisPoints, nil), []interface{}{&fastPriceFeed.MaxDeviationBasisPoints})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1MinAuthorizations, nil), []interface{}{&fastPriceFeed.MinAuthorizations})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1PriceDuration, nil), []interface{}{&fastPriceFeed.PriceDuration})
	rpcRequest.AddCall(callParamsFactory(fastPriceFeedMethodV1VolBasisPoints, nil), []interface{}{&fastPriceFeed.VolBasisPoints})

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
	}
	return err
}

func (r *FastPriceFeedV1Reader) readTokenData(
	ctx context.Context,
	address string,
	fastPriceFeed *FastPriceFeedV1,
	tokens []string,
) error {
	tokensLen := len(tokens)

	prices := make([]*big.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: fastPriceFeedMethodV1Prices,
			Params: []interface{}{common.HexToAddress(token)},
		}, []interface{}{&prices[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	for i, token := range tokens {
		fastPriceFeed.Prices[token] = prices[i]
	}

	return nil
}
