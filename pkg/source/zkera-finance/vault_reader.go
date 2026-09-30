package zkerafinance

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
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
			"liquiditySource": DexType,
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
	if validBlockNumber(vault.BlockNumber) {
		ctx = withBlockNumber(ctx, vault.BlockNumber)
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

func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := newRequest(r.ethrpcClient, ctx)

	rpcRequest.AddCall(callParamsFactory(vaultMethodHasDynamicFees, nil), []any{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []any{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIsSwapEnabled, nil), []any{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceFeed, nil), []any{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableSwapFeeBasisPoints, nil), []any{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableTaxBasisPoints, nil), []any{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodSwapFeeBasisPoints, nil), []any{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTaxBasisPoints, nil), []any{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTotalTokenWeights, nil), []any{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDG, nil), []any{&vault.USDGAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodWhitelistedTokenCount, nil), []any{&vault.WhitelistedTokensCount})

	response, err := rpcRequest.TryBlockAndAggregate()
	if err == nil && response != nil && validBlockNumber(response.BlockNumber) {
		vault.BlockNumber = response.BlockNumber
	}

	return err
}

func (r *VaultReader) readWhitelistedTokens(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokensLen := int(vault.WhitelistedTokensCount.Int64())

	whitelistedTokens := make([]common.Address, tokensLen)
	rpcRequest := newRequest(r.ethrpcClient, ctx)

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

func (r *VaultReader) readTokensData(
	ctx context.Context,
	address string,
	vault *Vault,
) error {
	tokensLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*uint256.Int, tokensLen)
	bufferAmounts := make([]*uint256.Int, tokensLen)
	reservedAmounts := make([]*uint256.Int, tokensLen)
	tokenDecimals := make([]*uint256.Int, tokensLen)
	stableTokens := make([]bool, tokensLen)
	usdgAmounts := make([]*uint256.Int, tokensLen)
	maxUSDGAmounts := make([]*uint256.Int, tokensLen)
	tokenWeights := make([]*uint256.Int, tokensLen)

	rpcRequest := newRequest(r.ethrpcClient, ctx)
	callParamsFactory := CallParamsFactory(r.abi, address)

	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultMethodPoolAmounts, []any{tokenAddress}), []any{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodBufferAmounts, []any{tokenAddress}), []any{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []any{tokenAddress}), []any{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenDecimals, []any{tokenAddress}), []any{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodStableTokens, []any{tokenAddress}), []any{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDGAmounts, []any{tokenAddress}), []any{&usdgAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDGAmounts, []any{tokenAddress}), []any{&maxUSDGAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenWeights, []any{tokenAddress}), []any{&tokenWeights[i]})
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		return err
	}

	for i, token := range vault.WhitelistedTokens {
		vault.PoolAmounts[token] = u256ToBig(poolAmounts[i])
		vault.BufferAmounts[token] = u256ToBig(bufferAmounts[i])
		vault.ReservedAmounts[token] = u256ToBig(reservedAmounts[i])
		vault.TokenDecimals[token] = u256ToBig(tokenDecimals[i])
		vault.StableTokens[token] = stableTokens[i]
		vault.USDGAmounts[token] = u256ToBig(usdgAmounts[i])
		vault.MaxUSDGAmounts[token] = u256ToBig(maxUSDGAmounts[i])
		vault.TokenWeights[token] = u256ToBig(tokenWeights[i])
	}

	return nil
}
