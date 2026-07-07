package quickperps

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
			"liquiditySource": DexTypeQuickperps,
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
//   - USDQAddress
//   - WhitelistedTokensCount
func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(vaultMethodHasDynamicFees, nil), []any{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []any{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIsSwapEnabled, nil), []any{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceFeed, nil), []any{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableSwapFeeBasisPoints, nil), []any{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableTaxBasisPoints, nil), []any{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodSwapFeeBasisPoints, nil), []any{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTaxBasisPoints, nil), []any{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTotalTokenWeights, nil), []any{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDQ, nil), []any{&vault.USDQAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodWhitelistedTokenCount, nil), []any{&vault.WhitelistedTokensCount})

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
			Params: []any{new(big.Int).SetInt64(int64(i))},
		}, []any{&whitelistedTokens[i]})
	}
	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	tokens := make([]string, tokensLen)
	for i := range whitelistedTokens {
		tokens[i] = hexutil.Encode(whitelistedTokens[i][:])
	}

	vault.WhitelistedTokens = tokens

	return nil
}

// readTokensData reads data which required token address as parameter, included:
// - PoolAmounts
// - TokenDecimals
// - StableTokens
// - USDQAmounts
// - MaxUSDQAmounts
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
	usdqAmounts := make([]*big.Int, tokensLen)
	maxUSDQAmounts := make([]*big.Int, tokensLen)
	tokenWeights := make([]*big.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	callParamsFactory := CallParamsFactory(r.abi, address)

	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultMethodPoolAmounts, []any{tokenAddress}), []any{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodBufferAmounts, []any{tokenAddress}), []any{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []any{tokenAddress}), []any{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenDecimals, []any{tokenAddress}), []any{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodStableTokens, []any{tokenAddress}), []any{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDQAmounts, []any{tokenAddress}), []any{&usdqAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDQAmounts, []any{tokenAddress}), []any{&maxUSDQAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenWeights, []any{tokenAddress}), []any{&tokenWeights[i]})
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
		vault.USDQAmounts[token] = usdqAmounts[i]
		vault.MaxUSDQAmounts[token] = maxUSDQAmounts[i]
		vault.TokenWeights[token] = tokenWeights[i]
	}

	return nil
}
