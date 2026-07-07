package madmex

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
			"liquiditySource": DexTypeMadmex,
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
// - BNB
// - BNBBUSD
// - BTC
// - BTCBNB
// - ChainlinkFlags
// - ETH
// - ETHBNB
// - FavorPrimaryPrice
// - IsAmmEnabled
// - IsSecondaryPriceEnabled
// - MaxStrictPriceDeviation
// - PriceSampleSpace
// - SecondaryPriceFeed
// - SpreadThresholdBasisPoints
// - UseV2Pricing
func (r *VaultPriceFeedReader) readData(
	ctx context.Context,
	address string,
	vaultPriceFeed *VaultPriceFeed,
) error {
	var bnb, btc, eth common.Address

	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodBNB, nil), []any{&bnb})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodBNBBUSD, nil), []any{&vaultPriceFeed.BNBBUSDAddress})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodBTC, nil), []any{&btc})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodBTCBNB, nil), []any{&vaultPriceFeed.BTCBNBAddress})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodChainlinkFlags, nil), []any{&vaultPriceFeed.ChainlinkFlagsAddress})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodETH, nil), []any{&eth})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodETHBNB, nil), []any{&vaultPriceFeed.ETHBNBAddress})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodFavorPrimaryPrice, nil), []any{&vaultPriceFeed.FavorPrimaryPrice})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodIsAmmEnabled, nil), []any{&vaultPriceFeed.IsAmmEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodIsSecondaryPriceEnabled, nil), []any{&vaultPriceFeed.IsSecondaryPriceEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodMaxStrictPriceDeviation, nil), []any{&vaultPriceFeed.MaxStrictPriceDeviation})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodPriceSampleSpace, nil), []any{&vaultPriceFeed.PriceSampleSpace})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodSecondaryPriceFeed, nil), []any{&vaultPriceFeed.SecondaryPriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodSpreadThresholdBasisPoints, nil), []any{&vaultPriceFeed.SpreadThresholdBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodUseV2Pricing, nil), []any{&vaultPriceFeed.UseV2Pricing})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggreate request: %s", err)
		return err
	}

	vaultPriceFeed.BNB = hexutil.Encode(bnb[:])
	vaultPriceFeed.BTC = hexutil.Encode(btc[:])
	vaultPriceFeed.ETH = hexutil.Encode(eth[:])

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
	tokensLen := len(tokens)

	priceFeedsAddresses := make([]common.Address, tokensLen)
	priceDecimals := make([]*big.Int, tokensLen)
	spreadBasisPoints := make([]*big.Int, tokensLen)
	adjustmentBasisPoints := make([]*big.Int, tokensLen)
	strictStableTokens := make([]bool, tokensLen)
	isAdjustmentAdditive := make([]bool, tokensLen)

	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodPriceFeeds, []any{tokenAddress}), []any{&priceFeedsAddresses[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodPriceDecimals, []any{tokenAddress}), []any{&priceDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodSpreadBasisPoints, []any{tokenAddress}), []any{&spreadBasisPoints[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodAdjustmentBasisPoints, []any{tokenAddress}), []any{&adjustmentBasisPoints[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodStrictStableTokens, []any{tokenAddress}), []any{&strictStableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodIsAdjustmentAdditive, []any{tokenAddress}), []any{&isAdjustmentAdditive[i]})
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
