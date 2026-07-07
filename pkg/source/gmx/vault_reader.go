package gmx

import (
	"context"
	"math/big"
	"strings"

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

	vaultMethodUSDG           string
	vaultMethodUSDGAmounts    string
	vaultMethodMaxUSDGAmounts string
}

func NewVaultReader(ethrpcClient *ethrpc.Client, usdgForkName string) *VaultReader {
	if usdgForkName == "" {
		usdgForkName = "usdg"
	}
	vaultABI, err := abi.JSON(strings.NewReader(replaceUsdgForkName(string(vaultJson), usdgForkName)))
	if err != nil {
		panic(err)
	}
	return &VaultReader{
		abi:          vaultABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmx,
			"reader":          "VaultReader",
		}),

		vaultMethodUSDG:           replaceUsdgForkName("usdg", usdgForkName),
		vaultMethodUSDGAmounts:    replaceUsdgForkName("usdgAmounts", usdgForkName),
		vaultMethodMaxUSDGAmounts: replaceUsdgForkName("maxUsdgAmounts", usdgForkName),
	}
}

func replaceUsdgForkName(s string, usdgForkName string) string {
	s = strings.ReplaceAll(s, "usdg", usdgForkName)
	s = strings.ReplaceAll(s, "Usdg", strings.ToUpper(usdgForkName[:1])+usdgForkName[1:])
	s = strings.ReplaceAll(s, "USDG", strings.ToUpper(usdgForkName))
	return s
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

	rpcRequest.AddCall(callParamsFactory(vaultMethodHasDynamicFees, nil), []any{&vault.HasDynamicFees})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIncludeAmmPrice, nil), []any{&vault.IncludeAmmPrice})
	rpcRequest.AddCall(callParamsFactory(vaultMethodIsSwapEnabled, nil), []any{&vault.IsSwapEnabled})
	rpcRequest.AddCall(callParamsFactory(vaultMethodPriceFeed, nil), []any{&vault.PriceFeedAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableSwapFeeBasisPoints, nil),
		[]any{&vault.StableSwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodStableTaxBasisPoints, nil),
		[]any{&vault.StableTaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodSwapFeeBasisPoints, nil), []any{&vault.SwapFeeBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTaxBasisPoints, nil), []any{&vault.TaxBasisPoints})
	rpcRequest.AddCall(callParamsFactory(vaultMethodTotalTokenWeights, nil), []any{&vault.TotalTokenWeights})
	rpcRequest.AddCall(callParamsFactory(r.vaultMethodUSDG, nil), []any{&vault.USDGAddress})
	rpcRequest.AddCall(callParamsFactory(vaultMethodAllWhitelistedTokensLength, nil),
		[]any{&vault.WhitelistedTokensCount})

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
			Method: vaultMethodAllWhitelistedTokens,
			Params: []any{new(big.Int).SetInt64(int64(i))},
		}, []any{&tokenList[i]})
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
			Params: []any{tokenList[i]},
		}, []any{&isWhitelistedTokens[i]})
	}
	_, err = rpcRequest.TryAggregate()
	if err != nil {
		return err
	}

	currentWhiteListTokens := make([]string, 0, tokensLen)
	for i := range tokenList {
		if isWhitelistedTokens[i] {
			currentWhiteListTokens = append(currentWhiteListTokens, hexutil.Encode(tokenList[i][:]))
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

		rpcRequest.AddCall(callParamsFactory(vaultMethodPoolAmounts, []any{tokenAddress}),
			[]any{&poolAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodBufferAmounts, []any{tokenAddress}),
			[]any{&bufferAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodReservedAmounts, []any{tokenAddress}),
			[]any{&reservedAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenDecimals, []any{tokenAddress}),
			[]any{&tokenDecimals[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodStableTokens, []any{tokenAddress}),
			[]any{&stableTokens[i]})
		rpcRequest.AddCall(callParamsFactory(r.vaultMethodUSDGAmounts, []any{tokenAddress}),
			[]any{&usdgAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(r.vaultMethodMaxUSDGAmounts, []any{tokenAddress}),
			[]any{&maxUSDGAmounts[i]})
		rpcRequest.AddCall(callParamsFactory(vaultMethodTokenWeights, []any{tokenAddress}),
			[]any{&tokenWeights[i]})
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
