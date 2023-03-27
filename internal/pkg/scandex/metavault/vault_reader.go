package metavault

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
)

type VaultReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewVaultReader(scanService *service.ScanService) *VaultReader {
	return &VaultReader{
		abi:         abis.MetavaultVault,
		scanService: scanService,
	}
}

// Read reads all data required for finding route
func (r *VaultReader) Read(ctx context.Context, address string) (*Vault, error) {
	vault := NewVault()

	if err := r.readData(ctx, address, vault); err != nil {
		return nil, err
	}

	if err := r.readWhitelistedTokens(ctx, address, vault); err != nil {
		return nil, err
	}

	if err := r.readTokensData(ctx, address, vault); err != nil {
		return nil, err
	}

	return vault, nil
}

// readData reads data which required no parameters, included:
//   - HasDynamicFees
//   - IsSwapEnabled
//   - PriceFeedAddress
//   - StableSwapFeeBasisPoints
//   - StableTaxBasisPoints
//   - SwapFeeBasisPoints
//   - TaxBasisPoints
//   - TotalTokenWeights
//   - USDMAddress
//   - WhitelistedTokensCount
func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	callParamsFactory := repository.CallParamsFactory(r.abi, address)
	calls := []*repository.CallParams{
		callParamsFactory(VaultMethodHasDynamicFees, &vault.HasDynamicFees, nil),
		callParamsFactory(VaultMethodIsSwapEnabled, &vault.IsSwapEnabled, nil),
		callParamsFactory(VaultMethodPriceFeed, &vault.PriceFeedAddress, nil),
		callParamsFactory(VaultMethodStableSwapFeeBasisPoints, &vault.StableSwapFeeBasisPoints, nil),
		callParamsFactory(VaultMethodStableTaxBasisPoints, &vault.StableTaxBasisPoints, nil),
		callParamsFactory(VaultMethodSwapFeeBasisPoints, &vault.SwapFeeBasisPoints, nil),
		callParamsFactory(VaultMethodTaxBasisPoints, &vault.TaxBasisPoints, nil),
		callParamsFactory(VaultMethodTotalTokenWeights, &vault.TotalTokenWeights, nil),
		callParamsFactory(VaultMethodUSDM, &vault.USDMAddress, nil),
		callParamsFactory(VaultMethodWhitelistedTokenCount, &vault.WhitelistedTokensCount, nil),
	}

	return r.scanService.MultiCall(ctx, calls)
}

// readWhitelistedTokens reads whitelistedTokens
func (r *VaultReader) readWhitelistedTokens(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	callParamsFactory := repository.CallParamsFactory(r.abi, address)

	tokensLen := int(vault.WhitelistedTokensCount.Int64())

	whitelistedTokens := make([]common.Address, tokensLen)
	var calls []*repository.CallParams

	for i := 0; i < tokensLen; i++ {
		calls = append(calls, callParamsFactory(
			VaultMethodAllWhitelistedTokens,
			&whitelistedTokens[i],
			[]interface{}{new(big.Int).SetInt64(int64(i))}),
		)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	tokens := make([]string, tokensLen)
	for i := range whitelistedTokens {
		tokens[i] = strings.ToLower(whitelistedTokens[i].String())
	}

	vault.WhitelistedTokens = tokens

	return nil
}

// readTokensData reads data which required token address as parameter, included:
// - PoolAmounts
// - TokenDecimals
// - StableTokens
// - USDMAmounts
// - MaxUSDMAmounts
// - TokenWeights
func (r *VaultReader) readTokensData(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	callParamsFactory := repository.CallParamsFactory(r.abi, address)
	tokenLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*big.Int, tokenLen)
	bufferAmounts := make([]*big.Int, tokenLen)
	reservedAmounts := make([]*big.Int, tokenLen)
	tokenDecimals := make([]*big.Int, tokenLen)
	stableTokens := make([]bool, tokenLen)
	usdmAmounts := make([]*big.Int, tokenLen)
	maxUSDMAmounts := make([]*big.Int, tokenLen)
	tokenWeights := make([]*big.Int, tokenLen)

	var calls []*repository.CallParams
	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		tokenCalls := []*repository.CallParams{
			callParamsFactory(VaultMethodPoolAmounts, &poolAmounts[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodBufferAmounts, &bufferAmounts[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodReservedAmounts, &reservedAmounts[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodTokenDecimals, &tokenDecimals[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodStableTokens, &stableTokens[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodUSDMAmounts, &usdmAmounts[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodMaxUSDMAmounts, &maxUSDMAmounts[i], []interface{}{tokenAddress}),
			callParamsFactory(VaultMethodTokenWeights, &tokenWeights[i], []interface{}{tokenAddress}),
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	for i, token := range vault.WhitelistedTokens {
		vault.PoolAmounts[token] = poolAmounts[i]
		vault.BufferAmounts[token] = bufferAmounts[i]
		vault.ReservedAmounts[token] = reservedAmounts[i]
		vault.TokenDecimals[token] = tokenDecimals[i]
		vault.StableTokens[token] = stableTokens[i]
		vault.USDMAmounts[token] = usdmAmounts[i]
		vault.MaxUSDMAmounts[token] = maxUSDMAmounts[i]
		vault.TokenWeights[token] = tokenWeights[i]
	}

	return nil
}
