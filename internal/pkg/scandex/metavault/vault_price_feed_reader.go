package metavault

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type VaultPriceFeedReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewVaultPriceFeedReader(scanService *service.ScanService) *VaultPriceFeedReader {
	return &VaultPriceFeedReader{
		abi:         abis.MetavaultVaultPriceFeed,
		scanService: scanService,
	}
}

func (r *VaultPriceFeedReader) Read(
	ctx context.Context,
	address string,
	tokens []string,
) (*VaultPriceFeed, error) {
	vaultPriceFeed := NewVaultPriceFeed()

	if err := r.readData(ctx, address, vaultPriceFeed); err != nil {
		return nil, err
	}

	if err := r.readTokenData(ctx, address, vaultPriceFeed, tokens); err != nil {
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
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	calls := []*repository.CallParams{
		callParamsFactory(VaultPriceFeedMethodIsSecondaryPriceEnabled, &vaultPriceFeed.IsSecondaryPriceEnabled, nil),
		callParamsFactory(VaultPriceFeedMethodMaxStrictPriceDeviation, &vaultPriceFeed.MaxStrictPriceDeviation, nil),
		callParamsFactory(VaultPriceFeedMethodPriceSampleSpace, &vaultPriceFeed.PriceSampleSpace, nil),
		callParamsFactory(VaultPriceFeedMethodSecondaryPriceFeed, &vaultPriceFeed.SecondaryPriceFeedAddress, nil),
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	tokenLen := len(tokens)

	priceFeedsAddresses := make([]common.Address, tokenLen)
	priceDecimals := make([]*big.Int, tokenLen)
	spreadBasisPoints := make([]*big.Int, tokenLen)
	adjustmentBasisPoints := make([]*big.Int, tokenLen)
	strictStableTokens := make([]bool, tokenLen)
	isAdjustmentAdditive := make([]bool, tokenLen)

	var calls []*repository.CallParams
	for i, token := range tokens {
		tokenAddress := common.HexToAddress(token)

		tokenCalls := []*repository.CallParams{
			callParamsFactory(VaultPriceFeedMethodPriceFeeds, &priceFeedsAddresses[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultPriceFeedMethodPriceDecimals, &priceDecimals[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultPriceFeedMethodSpreadBasisPoints, &spreadBasisPoints[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultPriceFeedMethodAdjustmentBasisPoints, &adjustmentBasisPoints[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultPriceFeedMethodStrictStableTokens, &strictStableTokens[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultPriceFeedMethodIsAdjustmentAdditive, &isAdjustmentAdditive[i], []interface{}{tokenAddress}),
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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
