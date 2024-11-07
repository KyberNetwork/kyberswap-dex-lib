package gmxglp

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type VaultReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewVaultReader(ethrpcClient *ethrpc.Client) *VaultReader {
	return &VaultReader{
		abi:          vaultABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmxGlp,
			"reader":          "VaultReader",
		}),
	}
}

// Read reads all data required for finding route
func (r *VaultReader) Read(ctx context.Context, address string) (*Vault, error) {
	vault := NewVault()

	if err := r.readData(ctx, address, vault); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, err
	}

	if err := r.readWhitelistedTokens(ctx, address, vault); err != nil {
		r.log.Errorf("error when read white listed token: %s", err)
		return nil, err
	}

	if err := r.readTokensData(ctx, address, vault); err != nil {
		r.log.Errorf("error when read tokens data: %s", err)
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
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(vaultMethodHasDynamicFees, nil), []interface{}{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []interface{}{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIsSwapEnabled, nil), []interface{}{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceFeed, nil), []interface{}{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableSwapFeeBasisPoints, nil), []interface{}{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableTaxBasisPoints, nil), []interface{}{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodSwapFeeBasisPoints, nil), []interface{}{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTaxBasisPoints, nil), []interface{}{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTotalTokenWeights, nil), []interface{}{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDG, nil), []interface{}{&vault.USDGAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodWhitelistedTokenCount, nil), []interface{}{&vault.WhitelistedTokensCount})
	rpcRequest.AddCall(callParamsFactory(vaultMethodMintBurnFeeBasisPoints, nil), []interface{}{&vault.MintBurnFeeBasicPoints})

	_, err := rpcRequest.TryAggregate()

	return err
}

// readWhitelistedTokens reads whitelistedTokens
func (r *VaultReader) readWhitelistedTokens(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokensLen := int(vault.WhitelistedTokensCount.Int64())

	whitelistedTokens := make([]common.Address, tokensLen)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < tokensLen; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vaultMethodAllWhitelistedTokens,
			Params: []interface{}{new(big.Int).SetInt64(int64(i))},
		}, []interface{}{&whitelistedTokens[i]})
	}
	if _, err := rpcRequest.TryAggregate(); err != nil {
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
	tokensLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*big.Int, tokensLen)
	bufferAmounts := make([]*big.Int, tokensLen)
	reservedAmounts := make([]*big.Int, tokensLen)
	tokenDecimals := make([]*big.Int, tokensLen)
	stableTokens := make([]bool, tokensLen)
	usdgAmounts := make([]*big.Int, tokensLen)
	maxUSDGAmounts := make([]*big.Int, tokensLen)
	tokenWeights := make([]*big.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	callParamsFactory := CallParamsFactory(r.abi, address)

	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultMethodPoolAmounts, []interface{}{tokenAddress}), []interface{}{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodBufferAmounts, []interface{}{tokenAddress}), []interface{}{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []interface{}{tokenAddress}), []interface{}{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenDecimals, []interface{}{tokenAddress}), []interface{}{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodStableTokens, []interface{}{tokenAddress}), []interface{}{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDGAmounts, []interface{}{tokenAddress}), []interface{}{&usdgAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDGAmounts, []interface{}{tokenAddress}), []interface{}{&maxUSDGAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenWeights, []interface{}{tokenAddress}), []interface{}{&tokenWeights[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
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
