package factory

import (
	"errors"
	"math/big"
	"strings"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// GetAddressesFromProvider to
// get registry, factory, getter addresses from Address Provider contract
func GetAddressesFromProvider(ctx context.Context, scanService *service.ScanService) ([]string, error) {
	var calls []*repository.CallParams
	// main, getter, meta, crypto registry, crypto factory
	addresses := make([]common.Address, 5)

	callParamsFactory := repository.CallParamsFactory(abis.CurveAddressProvider, AddressProvider)
	calls = append(
		calls,
		// This is the main registry address. Result contains both plain + lending + meta pools
		callParamsFactory("get_registry", &addresses[0], nil),

		// This is the Getter contract. Result [coins + underlying_coin + decimals + underlying_decimals], using for plain and lending pools
		callParamsFactory("get_address", &addresses[1], []interface{}{big.NewInt(1)}),

		// This is the Meta factory. Result infos for meta pools
		callParamsFactory("get_address", &addresses[2], []interface{}{big.NewInt(3)}),

		// This is the Crypto registry. Result infos for crypto pools (curve-v2)
		callParamsFactory("get_address", &addresses[3], []interface{}{big.NewInt(5)}),

		// This is the Crypto factory. Result infos for crypto-factory pools (curve-v2)
		callParamsFactory("get_address", &addresses[4], []interface{}{big.NewInt(6)}),
	)

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	res := make([]string, 0, len(addresses))
	for i := 0; i < len(addresses); i++ {
		res = append(res, addresses[i].Hex())
	}
	return res, nil
}

func GetPoolAddresses(
	ctx context.Context,
	scanService *service.ScanService,
	abi abi.ABI,
	target string,
	offset int64,
) ([]common.Address, int64, error) {
	if target == constant.AddressZero {
		return []common.Address{}, 0, nil
	}

	var calls []*repository.CallParams
	var poolCount *big.Int

	callParamsFactory := repository.CallParamsFactory(abi, target)
	calls = append(
		calls,
		callParamsFactory("pool_count", &poolCount, nil),
	)

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, 0, err
	}

	// Get Pool Addresses
	calls = make([]*repository.CallParams, 0, poolCount.Int64()-offset)
	var poolAddresses = make([]common.Address, poolCount.Int64()-offset)

	for i := offset; i < poolCount.Int64(); i++ {
		calls = append(
			calls,
			callParamsFactory("pool_list", &poolAddresses[i-offset], []interface{}{big.NewInt(int64(i))}),
		)
	}

	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, 0, err
	}

	return poolAddresses, poolCount.Int64(), nil
}

func GetAprecisions(ctx context.Context, scanService *service.ScanService, poolAddresses []common.Address) (
	[]*big.Int,
	error,
) {
	tryCalls := make([]*repository.TryCallParams, 0, 2*len(poolAddresses))

	var a = make([]*big.Int, len(poolAddresses))
	var aPrecises = make([]*big.Int, len(poolAddresses))

	for i := 0; i < len(poolAddresses); i++ {
		tryCallParamsFactory := repository.TryCallParamsFactory(abis.CurveMeta, poolAddresses[i].Hex())
		tryCalls = append(
			tryCalls,
			tryCallParamsFactory("A", &a[i], nil),
			tryCallParamsFactory("A_precise", &aPrecises[i], nil),
		)
	}

	// Execute try calls, do not require all success
	if err := scanService.TryAggregateForce(ctx, false, tryCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return nil, err
	}

	aPrecisions := make([]*big.Int, len(poolAddresses))
	for i := 0; i < len(poolAddresses); i++ {
		if a[i] != nil && aPrecises[i] != nil {
			aPrecisions[i] = new(big.Int).Div(aPrecises[i], a[i])
		} else if a[i] != nil {
			aPrecisions[i] = big.NewInt(1)
		} else {
			return nil, errors.New("cant get params A")
		}
	}
	return aPrecisions, nil
}

func IsPlainPool(tokens []string, underlyingTokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if !strings.EqualFold(underlyingTokens[i], AddressZero) && !strings.EqualFold(tokens[i], underlyingTokens[i]) {
			return false
		}
	}
	return true
}

func CommonAddressesToStrings(arr []common.Address) []string {
	var res []string
	for i := 0; i < len(arr); i++ {
		if arr[i].Hex() != AddressZero {
			res = append(res, strings.ToLower(arr[i].Hex()))
		}
	}
	return res
}

func IsExcluded(dex string) bool {
	for i := 0; i < len(IgnoreDexes); i++ {
		if strings.EqualFold(IgnoreDexes[i], dex) {
			return true
		}
	}
	return false
}
