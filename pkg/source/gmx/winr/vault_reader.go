package winr

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
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
			"liquiditySource": DexTypeWinr,
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

	if err := r.readWhitelistedTokensAndPriceFeedAddress(ctx, address, vault); err != nil {
		r.log.Errorf("error when read white listed token: %s", err)
		return nil, err
	}

	if err := r.readTokensData(ctx, address, vault); err != nil {
		r.log.Errorf("error when read tokens data: %s", err)
		return nil, err
	}

	return vault, nil
}

func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	callParamsFactory := gmx.CallParamsFactory(r.abi, address)
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodHasDynamicFees, nil), []interface{}{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodIsSwapEnabled, nil), []interface{}{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceOracleRouter, nil), []interface{}{&vault.PriceOracleAddress})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodStableSwapFeeBasisPoints, nil), []interface{}{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodStableTaxBasisPoints, nil), []interface{}{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodSwapFeeBasisPoints, nil), []interface{}{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodTaxBasisPoints, nil), []interface{}{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodTotalTokenWeights, nil), []interface{}{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDW, nil), []interface{}{&vault.USDWAddress})
	rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodAllWhitelistedTokensLength, nil), []interface{}{&vault.WhitelistedTokensCount})

	_, err := rpcRequest.TryAggregate()

	return err
}

// readWhitelistedTokens reads whitelistedTokens
func (r *VaultReader) readWhitelistedTokensAndPriceFeedAddress(
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
			Method: gmx.VaultMethodAllWhitelistedTokens,
			Params: []interface{}{new(big.Int).SetInt64(int64(i))},
		}, []interface{}{&tokenList[i]})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    priceOracleRouterABI,
		Target: vault.PriceOracleAddress.String(),
		Method: vaultMethodPrimaryPriceFeed,
		Params: nil,
	}, []interface{}{&vault.PriceFeedAddress})
	_, err := rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	currentWhiteListTokens := make([]string, 0, tokensLen)
	for i := range tokenList {
		currentWhiteListTokens = append(currentWhiteListTokens, strings.ToLower(tokenList[i].String()))
	}

	vault.WhitelistedTokens = currentWhiteListTokens

	return nil
}

func (r *VaultReader) readTokensData(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokensLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*big.Int, tokensLen)
	bufferAmounts := make([]*big.Int, tokensLen)
	// reservedAmounts := make([]*big.Int, tokensLen)
	tokenDecimals := make([]*big.Int, tokensLen)
	stableTokens := make([]bool, tokensLen)
	usdgAmounts := make([]*big.Int, tokensLen)
	maxUSDGAmounts := make([]*big.Int, tokensLen)
	tokenWeights := make([]*big.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	callParamsFactory := gmx.CallParamsFactory(r.abi, address)

	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodPoolAmount, []interface{}{tokenAddress}), []interface{}{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodBufferAmounts, []interface{}{tokenAddress}), []interface{}{&bufferAmounts[i]})
		// rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []interface{}{tokenAddress}), []interface{}{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodTokenDecimals, []interface{}{tokenAddress}), []interface{}{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodStableTokens, []interface{}{tokenAddress}), []interface{}{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDWAmounts, []interface{}{tokenAddress}), []interface{}{&usdgAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDWAmounts, []interface{}{tokenAddress}), []interface{}{&maxUSDGAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(gmx.VaultMethodTokenWeights, []interface{}{tokenAddress}), []interface{}{&tokenWeights[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	for i, token := range vault.WhitelistedTokens {
		vault.PoolAmounts[token] = poolAmounts[i]
		vault.BufferAmounts[token] = bufferAmounts[i]
		// vault.ReservedAmounts[token] = reservedAmounts[i]
		vault.TokenDecimals[token] = tokenDecimals[i]
		vault.StableTokens[token] = stableTokens[i]
		vault.USDWAmounts[token] = usdgAmounts[i]
		vault.MaxUSDWAmounts[token] = maxUSDGAmounts[i]
		vault.TokenWeights[token] = tokenWeights[i]
	}

	return nil
}
