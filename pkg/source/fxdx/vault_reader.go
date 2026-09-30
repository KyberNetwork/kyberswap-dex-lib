package fxdx

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
			"liquiditySource": DexTypeFxdx,
			"reader":          "VaultReader",
		}),
	}
}

func (r *VaultReader) Read(ctx context.Context, address string) (*Vault, *big.Int, error) {
	vault := NewVault()

	if err := r.readData(ctx, address, vault); err != nil {
		r.log.Errorf("error when read data: %s", err)
		return nil, nil, err
	}

	if err := r.readWhitelistedTokens(ctx, address, vault); err != nil {
		r.log.Errorf("error when read white listed token: %s", err)
		return nil, nil, err
	}

	blockNumber, err := r.readTokensData(ctx, address, vault)
	if err != nil {
		r.log.Errorf("error when read tokens data: %s", err)
		return nil, nil, err
	}

	return vault, blockNumber, nil
}

func (r *VaultReader) readData(ctx context.Context, address string, vault *Vault) error {
	callParamsFactory := CallParamsFactory(r.abi, address)
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []any{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIsSwapEnabled, nil), []any{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceFeed, nil), []any{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTotalTokenWeights, nil), []any{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(vaultMethodUSDF, nil), []any{&vault.USDFAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodWhitelistedTokenCount, nil), []any{&vault.WhitelistedTokensCount})
	rpcRequest.AddCall(callParamsFactory(vaultMethodFeeUtils, nil), []any{&vault.FeeUtils})

	_, err := rpcRequest.TryAggregate()

	return err
}

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

func (r *VaultReader) readTokensData(ctx context.Context, address string, vault *Vault) (*big.Int, error) {
	tokensLen := len(vault.WhitelistedTokens)
	poolAmounts := make([]*uint256.Int, tokensLen)
	bufferAmounts := make([]*uint256.Int, tokensLen)
	reservedAmounts := make([]*uint256.Int, tokensLen)
	tokenDecimals := make([]*uint256.Int, tokensLen)
	stableTokens := make([]bool, tokensLen)
	usdfAmounts := make([]*uint256.Int, tokensLen)
	maxUSDFAmounts := make([]*uint256.Int, tokensLen)
	tokenWeights := make([]*uint256.Int, tokensLen)

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)
	callParamsFactory := CallParamsFactory(r.abi, address)

	for i, token := range vault.WhitelistedTokens {
		tokenAddress := common.HexToAddress(token)

		rpcRequest.AddCall(callParamsFactory(vaultMethodPoolAmounts, []any{tokenAddress}), []any{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodBufferAmounts, []any{tokenAddress}), []any{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []any{tokenAddress}), []any{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenDecimals, []any{tokenAddress}), []any{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodStableTokens, []any{tokenAddress}), []any{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodUSDFAmounts, []any{tokenAddress}), []any{&usdfAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodMaxUSDFAmounts, []any{tokenAddress}), []any{&maxUSDFAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenWeights, []any{tokenAddress}), []any{&tokenWeights[i]})
	}

	response, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}

	for i, token := range vault.WhitelistedTokens {
		vault.PoolAmounts[token] = u256ToBig(poolAmounts[i])
		vault.BufferAmounts[token] = u256ToBig(bufferAmounts[i])
		vault.ReservedAmounts[token] = u256ToBig(reservedAmounts[i])
		vault.TokenDecimals[token] = u256ToBig(tokenDecimals[i])
		vault.StableTokens[token] = stableTokens[i]
		vault.USDFAmounts[token] = u256ToBig(usdfAmounts[i])
		vault.MaxUSDFAmounts[token] = u256ToBig(maxUSDFAmounts[i])
		vault.TokenWeights[token] = u256ToBig(tokenWeights[i])
	}

	return response.BlockNumber, nil
}
