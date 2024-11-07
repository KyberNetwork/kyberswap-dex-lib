package fulcrom

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
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
			"liquiditySource": DexTypeFulcrom,
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

	if err := r.readTokenData(ctx, address, vaultPriceFeed, tokens); err != nil {
		r.log.Errorf("error when read token data: %s", err)
		return nil, err
	}

	return vaultPriceFeed, nil
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

	maxPrices := make([]*big.Int, tokensLen)
	minPrices := make([]*big.Int, tokensLen)

	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, token := range tokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodGetPrice, []interface{}{tokenAddress, true}), []interface{}{&maxPrices[i]})
		rpcRequest.AddCall(callParamsFactory(vaultPriceFeedMethodGetPrice, []interface{}{tokenAddress, false}), []interface{}{&minPrices[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		r.log.Errorf("error when call aggregate request: %s", err)
		return err
	}

	for i, token := range tokens {
		vaultPriceFeed.MinPrices[token] = minPrices[i]
		vaultPriceFeed.MaxPrices[token] = maxPrices[i]
	}

	return nil
}
