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

type VaultReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewVaultReader(scanService *service.ScanService) *VaultReader {
	return &VaultReader{
		abi:         abis.GMXVault,
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
//   - IncludeAmmPrice
//   - IsSwapEnabled
//   - PriceFeedAddress
//   - StableSwapFeeBasisPoints
//   - StableTaxBasisPoints
//   - SwapFeeBasisPoints
//   - TaxBasisPoints
//   - TotalTokenWeights
//   - USDGAddress
//   - WhitelistedTokensCount
func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodHasDynamicFees,
			Params: nil,
			Output: &vault.HasDynamicFees,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodIncludeAmmPrice,
			Params: nil,
			Output: &vault.IncludeAmmPrice,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodIsSwapEnabled,
			Params: nil,
			Output: &vault.IsSwapEnabled,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodPriceFeed,
			Params: nil,
			Output: &vault.PriceFeedAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodStableSwapFeeBasisPoints,
			Params: nil,
			Output: &vault.StableSwapFeeBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodStableTaxBasisPoints,
			Params: nil,
			Output: &vault.StableTaxBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodSwapFeeBasisPoints,
			Params: nil,
			Output: &vault.SwapFeeBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodTaxBasisPoints,
			Params: nil,
			Output: &vault.TaxBasisPoints,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodTotalTokenWeights,
			Params: nil,
			Output: &vault.TotalTokenWeights,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodUSDG,
			Params: nil,
			Output: &vault.USDGAddress,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodWhitelistedTokenCount,
			Params: nil,
			Output: &vault.WhitelistedTokensCount,
		},
	}

	return r.scanService.MultiCall(ctx, calls)
}

// readWhitelistedTokens reads whitelistedTokens
func (r *VaultReader) readWhitelistedTokens(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokensLen := int(vault.WhitelistedTokensCount.Int64())

	whitelistedTokens := make([]common.Address, tokensLen)
	var calls []*repository.CallParams

	for i := 0; i < tokensLen; i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodAllWhitelistedTokens,
			Params: []interface{}{new(big.Int).SetInt64(int64(i))},
			Output: &whitelistedTokens[i],
		})
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
// - USDGAmounts
// - MaxUSDGAmounts
// - TokenWeights
func (r *VaultReader) readTokensData(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokenLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*big.Int, tokenLen)
	bufferAmounts := make([]*big.Int, tokenLen)
	reservedAmounts := make([]*big.Int, tokenLen)
	tokenDecimals := make([]*big.Int, tokenLen)
	stableTokens := make([]bool, tokenLen)
	usdgAmounts := make([]*big.Int, tokenLen)
	maxUSDGAmounts := make([]*big.Int, tokenLen)
	tokenWeights := make([]*big.Int, tokenLen)

	var calls []*repository.CallParams
	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		tokenCalls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodPoolAmounts,
				Params: []interface{}{tokenAddress},
				Output: &poolAmounts[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodBufferAmounts,
				Params: []interface{}{tokenAddress},
				Output: &bufferAmounts[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodReservedAmounts,
				Params: []interface{}{tokenAddress},
				Output: &reservedAmounts[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodTokenDecimals,
				Params: []interface{}{tokenAddress},
				Output: &tokenDecimals[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodStableTokens,
				Params: []interface{}{tokenAddress},
				Output: &stableTokens[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodUSDGAmounts,
				Params: []interface{}{tokenAddress},
				Output: &usdgAmounts[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodMaxUSDGAmounts,
				Params: []interface{}{tokenAddress},
				Output: &maxUSDGAmounts[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: VaultMethodTokenWeights,
				Params: []interface{}{tokenAddress},
				Output: &tokenWeights[i],
			},
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
		vault.USDGAmounts[token] = usdgAmounts[i]
		vault.MaxUSDGAmounts[token] = maxUSDGAmounts[i]
		vault.TokenWeights[token] = tokenWeights[i]
	}

	return nil
}
