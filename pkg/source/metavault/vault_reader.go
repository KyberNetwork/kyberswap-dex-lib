package metavault

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
			"liquiditySource": DexTypeMetavault,
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
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(VaultMethodHasDynamicFees, nil), []any{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(VaultMethodIsSwapEnabled, nil), []any{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(VaultMethodPriceFeed, nil), []any{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(VaultMethodStableSwapFeeBasisPoints, nil), []any{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodStableTaxBasisPoints, nil), []any{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodSwapFeeBasisPoints, nil), []any{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodTaxBasisPoints, nil), []any{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(VaultMethodTotalTokenWeights, nil), []any{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(VaultMethodUSDM, nil), []any{&vault.USDMAddress})
	rpcRequest.AddCall(callParamsFactory(VaultMethodWhitelistedTokenCount, nil), []any{&vault.WhitelistedTokensCount})

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
			Method: VaultMethodAllWhitelistedTokens,
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
// - USDMAmounts
// - MaxUSDMAmounts
// - TokenWeights
func (r *VaultReader) readTokensData(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	tokensLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*big.Int, tokensLen)
	bufferAmounts := make([]*big.Int, tokensLen)
	reservedAmounts := make([]*big.Int, tokensLen)
	tokenDecimals := make([]*big.Int, tokensLen)
	stableTokens := make([]bool, tokensLen)
	usdmAmounts := make([]*big.Int, tokensLen)
	maxUSDMAmounts := make([]*big.Int, tokensLen)
	tokenWeights := make([]*big.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(VaultMethodPoolAmounts, []any{tokenAddress}), []any{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodBufferAmounts, []any{tokenAddress}), []any{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodReservedAmounts, []any{tokenAddress}), []any{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodTokenDecimals, []any{tokenAddress}), []any{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodStableTokens, []any{tokenAddress}), []any{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodUSDMAmounts, []any{tokenAddress}), []any{&usdmAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodMaxUSDMAmounts, []any{tokenAddress}), []any{&maxUSDMAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(VaultMethodTokenWeights, []any{tokenAddress}), []any{&tokenWeights[i]})
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
		vault.USDMAmounts[token] = usdmAmounts[i]
		vault.MaxUSDMAmounts[token] = maxUSDMAmounts[i]
		vault.TokenWeights[token] = tokenWeights[i]
	}

	return nil
}
