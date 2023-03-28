package synthetix

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
)

const (
	// Synthetix methods

	PoolStateMethodAvailableCurrencyKeys        = "availableCurrencyKeys"
	PoolStateMethodAvailableSynthCount          = "availableSynthCount"
	PoolStateMethodAvailableSynths              = "availableSynths"
	PoolStateMethodGetSUSDCurrencyKey           = "sUSD"
	PoolStateMethodGetSynthAddressByCurrencyKey = "synths"
	PoolStateMethodTotalIssuedSynths            = "totalIssuedSynths" // to get the "reserves" of tokens

	// MultiCollateralSynth methods

	MultiCollateralSynthMethodGetProxy = "proxy"

	// ProxyERC20 methods

	ProxyERC20MethodTotalSupply = "totalSupply"
)

type PoolStateReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewPoolStateReader(scanService *service.ScanService) *PoolStateReader {
	return &PoolStateReader{
		abi:         abis.Synthetix,
		scanService: scanService,
	}
}

// Read reads all data required for finding route
func (r *PoolStateReader) Read(ctx context.Context, address string) (*PoolState, error) {
	poolState := NewPoolState()

	if err := r.readBlockNumber(ctx, poolState); err != nil {
		return nil, err
	}

	if err := r.readData(ctx, address, poolState); err != nil {
		return nil, err
	}

	if err := r.readSynthTokens(ctx, address, poolState); err != nil {
		return nil, err
	}

	return poolState, nil
}

func (r *PoolStateReader) readBlockNumber(
	ctx context.Context,
	poolState *PoolState,
) error {
	latestBlockTimestamp, err := r.scanService.GetLatestBlockTimestamp(ctx)
	if err != nil {
		return err
	}

	poolState.BlockTimestamp = latestBlockTimestamp

	return nil
}

// readData reads data which required no parameters, included:
//   - CurrencyKeys
//   - AvailableSynthCount
//   - sUSD
//   - TotalIssuedSUSD
func (r *PoolStateReader) readData(ctx context.Context, address string, poolState *PoolState) error {
	var currencyKeysResult [][32]byte
	var sUSDResult [32]byte
	var totalIssuedSUSD *big.Int

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodAvailableCurrencyKeys,
			Params: nil,
			Output: &currencyKeysResult,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodAvailableSynthCount,
			Params: nil,
			Output: &poolState.AvailableSynthCount,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodGetSUSDCurrencyKey,
			Params: nil,
			Output: &sUSDResult,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	currencyKeys := make([]string, len(currencyKeysResult))
	for i, key := range currencyKeysResult {
		currencyKeys[i] = common.BytesToHash(key[:]).String()
	}
	poolState.CurrencyKeys = currencyKeys

	poolState.SUSDCurrencyKey = common.BytesToHash(sUSDResult[:]).String()

	if err := r.scanService.Call(ctx, &repository.CallParams{
		ABI:    r.abi,
		Target: address,
		Method: PoolStateMethodTotalIssuedSynths,
		Params: []interface{}{sUSDResult},
		Output: &totalIssuedSUSD,
	}); err != nil {
		return err
	}

	poolState.TotalIssuedSUSD = totalIssuedSUSD

	return nil
}

// readSynthTokens reads synths data and require currencyKey parameter, included:
//   - Synths
//   - SynthsTotalSupply
func (r *PoolStateReader) readSynthTokens(
	ctx context.Context,
	address string,
	poolState *PoolState,
) error {
	synthsLen := int(poolState.AvailableSynthCount.Int64())

	currencyKeys := poolState.CurrencyKeys
	currencyKeysLen := len(currencyKeys)
	synths := make([]common.Address, currencyKeysLen)
	var synthCalls []*repository.CallParams

	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		calls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: PoolStateMethodGetSynthAddressByCurrencyKey,
				Params: []interface{}{keyByte},
				Output: &synths[i],
			},
		}

		synthCalls = append(synthCalls, calls...)
	}

	if err := r.scanService.MultiCall(ctx, synthCalls); err != nil {
		return err
	}

	synthProxyResults := make([]common.Address, synthsLen)
	var proxyCalls []*repository.CallParams

	for i, synthAddress := range synths {
		calls := []*repository.CallParams{
			{
				ABI:    abis.SynthetixMultiCollateralSynth,
				Target: synthAddress.String(),
				Method: MultiCollateralSynthMethodGetProxy,
				Params: nil,
				Output: &synthProxyResults[i],
			},
		}

		proxyCalls = append(proxyCalls, calls...)
	}

	if err := r.scanService.MultiCall(ctx, proxyCalls); err != nil {
		return err
	}

	totalSupply := make([]*big.Int, synthsLen)
	var totalSupplyCalls []*repository.CallParams

	for i, proxyAddress := range synthProxyResults {
		calls := []*repository.CallParams{
			{
				ABI:    abis.SynthetixMultiCollateralSynth,
				Target: proxyAddress.String(),
				Method: ProxyERC20MethodTotalSupply,
				Params: nil,
				Output: &totalSupply[i],
			},
		}

		totalSupplyCalls = append(totalSupplyCalls, calls...)
	}

	if err := r.scanService.MultiCall(ctx, totalSupplyCalls); err != nil {
		return err
	}

	for i, key := range currencyKeys {
		poolState.Synths[key] = synthProxyResults[i]
		poolState.SynthsTotalSupply[key] = totalSupply[i]
		poolState.CurrencyKeyBySynth[synthProxyResults[i]] = key
	}

	return nil
}
