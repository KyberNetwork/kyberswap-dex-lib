package metavault

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type VaultPriceFeedReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewVaultPriceFeedReader(ethrpcClient *ethrpc.Client) *VaultPriceFeedReader {
	return &VaultPriceFeedReader{
		abi:          vaultPriceFeedABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeMetavault,
			"reader":          "VaultPriceFeedReader",
		}),
	}
}

func (r *VaultPriceFeedReader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*VaultPriceFeed, error) {
	vaultPriceFeed := NewVaultPriceFeed()

	if err := r.readData(ctx, address, vaultPriceFeed); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, err
	}

	if err := r.readTokenData(ctx, address, vaultPriceFeed, tokens); err != nil {
		r.log.Errorf("error when read token data: %s", err)
		return nil, err
	}

	return vaultPriceFeed, nil
}

// readData reads data which required no parameters, included:
// - IsSecondaryPriceEnabled
// - MaxStrictPriceDeviation
// - PriceSampleSpace
// - SecondaryPriceFeed
func (r *VaultPriceFeedReader) readData(
	ctx context.Context,
	address string,
	vaultPriceFeed *VaultPriceFeed,
) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodIsSecondaryPriceEnabled, nil), []interface{}{&vaultPriceFeed.IsSecondaryPriceEnabled})
	rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodMaxStrictPriceDeviation, nil), []interface{}{&vaultPriceFeed.MaxStrictPriceDeviation})
	rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodPriceSampleSpace, nil), []interface{}{&vaultPriceFeed.PriceSampleSpace})
	rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodSecondaryPriceFeed, nil), []interface{}{&vaultPriceFeed.SecondaryPriceFeedAddress})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
		return err
	}

	return nil
}

// readTokenData reads data which required token address as parameter, included:
// - PriceFeedsAddresses
// - PriceDecimals
// - SpreadBasisPoints
// - AdjustmentBasisPoints
// - StrictStableTokens
// - IsAdjustmentAdditive
func (r *VaultPriceFeedReader) readTokenData(
	ctx context.Context,
	address string,
	vaultPriceFeed *VaultPriceFeed,
	tokens []string,
) error {
	callParamsFactory := CallParamsFactory(r.abi, address)

	tokensLen := len(tokens)

	priceFeedsAddresses := make([]common.Address, tokensLen)
	priceDecimals := make([]*big.Int, tokensLen)
	spreadBasisPoints := make([]*big.Int, tokensLen)
	adjustmentBasisPoints := make([]*big.Int, tokensLen)
	strictStableTokens := make([]bool, tokensLen)
	isAdjustmentAdditive := make([]bool, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, token := range tokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodPriceFeeds, []interface{}{tokenAddress}), []interface{}{&priceFeedsAddresses[i]})
		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodPriceDecimals, []interface{}{tokenAddress}), []interface{}{&priceDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodSpreadBasisPoints, []interface{}{tokenAddress}), []interface{}{&spreadBasisPoints[i]})
		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodAdjustmentBasisPoints, []interface{}{tokenAddress}), []interface{}{&adjustmentBasisPoints[i]})
		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodStrictStableTokens, []interface{}{tokenAddress}), []interface{}{&strictStableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(VaultPriceFeedMethodIsAdjustmentAdditive, []interface{}{tokenAddress}), []interface{}{&isAdjustmentAdditive[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
		return err
	}

	for i, token := range tokens {
		vaultPriceFeed.PriceFeedsAddresses[token] = priceFeedsAddresses[i]
		vaultPriceFeed.PriceDecimals[token] = priceDecimals[i]
		vaultPriceFeed.SpreadBasisPoints[token] = spreadBasisPoints[i]
		vaultPriceFeed.AdjustmentBasisPoints[token] = adjustmentBasisPoints[i]
		vaultPriceFeed.StrictStableTokens[token] = strictStableTokens[i]
		vaultPriceFeed.IsAdjustmentAdditive[token] = isAdjustmentAdditive[i]
	}

	return nil
}
