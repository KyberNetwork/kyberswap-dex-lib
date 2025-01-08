package gmx

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
			"liquiditySource": DexTypeGmx,
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

	rpcRequest.AddCall(callParamsFactory(VaultMethodHasDynamicFees, nil), []interface{}{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []interface{}{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(VaultMethodIsSwapEnabled, nil), []interface{}{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(VaultMethodPriceFeed, nil), []interface{}{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(VaultMethodStableSwapFeeBasisPoints, nil), []interface{}{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodStableTaxBasisPoints, nil), []interface{}{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodSwapFeeBasisPoints, nil), []interface{}{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodTaxBasisPoints, nil), []interface{}{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodTotalTokenWeights, nil), []interface{}{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDG, nil), []interface{}{&vault.USDGAddress})
	rpcRequest.AddCall(callParamsFactory(VaultMethodAllWhitelistedTokensLength, nil), []interface{}{&vault.WhitelistedTokensCount})

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

	tokenList := make([]common.Address, tokensLen)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < tokensLen; i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: VaultMethodAllWhitelistedTokens,
			Params: []interface{}{new(big.Int).SetInt64(int64(i))},
		}, []interface{}{&tokenList[i]})
	}
	res, err := rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	isWhitelistedTokens := make([]bool, len(tokenList))
	rpcRequest = r.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)

	for i := 0; i < len(tokenList); i++ {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: vaultMethodWhitelistedTokens,
			Params: []interface{}{tokenList[i]},
		}, []interface{}{&isWhitelistedTokens[i]})
	}
	_, err = rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	currentWhiteListTokens := make([]string, 0, tokensLen)
	for i := range tokenList {
		if isWhitelistedTokens[i] {
			currentWhiteListTokens = append(currentWhiteListTokens, strings.ToLower(tokenList[i].String()))
		}
	}

	vault.WhitelistedTokens = currentWhiteListTokens

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

		rpcRequest.AddCall(callParamsFactory(VaultMethodPoolAmount, []interface{}{tokenAddress}), []interface{}{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodBufferAmounts, []interface{}{tokenAddress}), []interface{}{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []interface{}{tokenAddress}), []interface{}{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodTokenDecimals, []interface{}{tokenAddress}), []interface{}{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodStableTokens, []interface{}{tokenAddress}), []interface{}{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDGAmounts, []interface{}{tokenAddress}), []interface{}{&usdgAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDGAmounts, []interface{}{tokenAddress}), []interface{}{&maxUSDGAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodTokenWeights, []interface{}{tokenAddress}), []interface{}{&tokenWeights[i]})
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
