package madmex

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type VaultPriceFeedReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewVaultPriceFeedReader(scanService *service.ScanService) *VaultPriceFeedReader {
	return &VaultPriceFeedReader{
		abi:         abis.GMXVaultPriceFeed,
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

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodBNB,
			Params: nil,
			Output: &bnb,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodBNBBUSD,
			Params: nil,
			Output: &vaultPriceFeed.BNBBUSDAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodBTC,
			Params: nil,
			Output: &btc,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodBTCBNB,
			Params: nil,
			Output: &vaultPriceFeed.BTCBNBAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodChainlinkFlags,
			Params: nil,
			Output: &vaultPriceFeed.ChainlinkFlagsAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodETH,
			Params: nil,
			Output: &eth,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodETHBNB,
			Params: nil,
			Output: &vaultPriceFeed.ETHBNBAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodFavorPrimaryPrice,
			Params: nil,
			Output: &vaultPriceFeed.FavorPrimaryPrice,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodIsAmmEnabled,
			Params: nil,
			Output: &vaultPriceFeed.IsAmmEnabled,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodIsSecondaryPriceEnabled,
			Params: nil,
			Output: &vaultPriceFeed.IsSecondaryPriceEnabled,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodMaxStrictPriceDeviation,
			Params: nil,
			Output: &vaultPriceFeed.MaxStrictPriceDeviation,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodPriceSampleSpace,
			Params: nil,
			Output: &vaultPriceFeed.PriceSampleSpace,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodSecondaryPriceFeed,
			Params: nil,
			Output: &vaultPriceFeed.SecondaryPriceFeedAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodSpreadThresholdBasisPoints,
			Params: nil,
			Output: &vaultPriceFeed.SpreadThresholdBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultPriceFeedMethodUseV2Pricing,
			Params: nil,
			Output: &vaultPriceFeed.UseV2Pricing,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	vaultPriceFeed.BNB = strings.ToLower(bnb.String())
	vaultPriceFeed.BTC = strings.ToLower(btc.String())
	vaultPriceFeed.ETH = strings.ToLower(eth.String())

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
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodPriceFeeds,
				Params: []interface{}{tokenAddress},
				Output: &priceFeedsAddresses[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodPriceDecimals,
				Params: []interface{}{tokenAddress},
				Output: &priceDecimals[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodSpreadBasisPoints,
				Params: []interface{}{tokenAddress},
				Output: &spreadBasisPoints[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodAdjustmentBasisPoints,
				Params: []interface{}{tokenAddress},
				Output: &adjustmentBasisPoints[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodStrictStableTokens,
				Params: []interface{}{tokenAddress},
				Output: &strictStableTokens[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultPriceFeedMethodIsAdjustmentAdditive,
				Params: []interface{}{tokenAddress},
				Output: &isAdjustmentAdditive[i],
			},
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
