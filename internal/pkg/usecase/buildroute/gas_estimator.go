package buildroute

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
)

const (
	routerABI = `[{
		"inputs": [{
			"components": [
				{"internalType": "address", "name": "callTarget", "type": "address"},
				{"internalType": "address", "name": "approveTarget", "type": "address"},
				{"internalType": "bytes", "name": "targetData", "type": "bytes"},
				{"components": [
					{"internalType": "contract IERC20", "name": "srcToken", "type": "address"},
					{"internalType": "contract IERC20", "name": "dstToken", "type": "address"},
					{"internalType": "address", "name": "dstReceiver", "type": "address"},
					{"internalType": "uint256", "name": "amount", "type": "uint256"},
					{"internalType": "uint256", "name": "minReturnAmount", "type": "uint256"},
					{"internalType": "uint256", "name": "flags", "type": "uint256"}
				], "internalType": "struct MetaAggregationRouterV2.SwapDescriptionV2", "name": "desc", "type": "tuple"},
				{"internalType": "bytes", "name": "clientData", "type": "bytes"}
			], "internalType": "struct MetaAggregationRouterV2.SwapExecutionParams", "name": "execution", "type": "tuple"}
		],
		"name": "swap",
		"outputs": [
			{"internalType": "uint256", "name": "returnAmount", "type": "uint256"},
			{"internalType": "uint256", "name": "gasUsed", "type": "uint256"}
		],
		"stateMutability": "payable",
		"type": "function"
	}]`
)

type UnsignedTransaction struct {
	sender   string
	data     string
	value    *big.Int
	gasPrice *big.Int
}

type GasEstimator struct {
	ethClient              IETHClient
	gasRepository          IGasRepository
	onchainpriceRepository IOnchainPriceRepository

	routerAddress   string
	routerABI       abi.ABI
	gasTokenAddress string
}

func NewGasEstimator(
	ethClient IETHClient,
	gasRepo IGasRepository,
	onchainPriceRepository IOnchainPriceRepository,
	chainId valueobject.ChainID,
	routerAddress string,
) *GasEstimator {
	parsedABI, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		panic(err)
	}

	return &GasEstimator{
		ethClient:              ethClient,
		gasRepository:          gasRepo,
		onchainpriceRepository: onchainPriceRepository,
		routerAddress:          routerAddress,
		routerABI:              parsedABI,
		gasTokenAddress:        strings.ToLower(valueobject.WrappedNativeMap[chainId]),
	}
}

func (e *GasEstimator) EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, *big.Int, error) {
	callMsg, err := e.prepareCallMsg(tx)
	if err != nil {
		return 0, nil, err
	}

	gasUsed, returnAmount, err := e.simulateSwap(ctx, callMsg)

	clientid := clientid.GetClientIDFromCtx(ctx)
	metrics.CountEstimateGas(ctx, err == nil, "allDexes", clientid)
	return gasUsed, returnAmount, err
}

// EstimateGasAndPriceUSD performs a complete gas estimation including price calculations
// Returns:
// - estimatedGas: the estimated gas required for the transaction
// - priceInUSD: the estimated gas cost in USD
// - returnAmount: the actual output amount from the swap (if simulation successful)
// - error: any error that occurred during estimation
func (e *GasEstimator) EstimateGasAndPriceUSD(ctx context.Context, tx UnsignedTransaction) (uint64, float64, *big.Int, error) {
	estimatedGas, returnAmount, err := e.EstimateGas(ctx, tx)
	if err != nil {
		return 0, 0.0, nil, err
	}

	gasPrice, err := e.getGasPrice(ctx)
	if err != nil {
		return 0, 0.0, nil, err
	}

	gasTokenPriceUSD, err := e.GetGasTokenPriceUSD(ctx)
	if err != nil {
		return 0, 0.0, nil, err
	}

	priceInUSD := utils.CalcGasUsd(gasPrice, int64(estimatedGas), gasTokenPriceUSD)
	return estimatedGas, priceInUSD, returnAmount, nil
}

func (e *GasEstimator) prepareCallMsg(tx UnsignedTransaction) (ethereum.CallMsg, error) {
	encodedData, err := hexutil.Decode(tx.data)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	to := common.HexToAddress(e.routerAddress)
	return ethereum.CallMsg{
		From:     common.HexToAddress(tx.sender),
		To:       &to,
		GasPrice: tx.gasPrice,
		Value:    tx.value,
		Data:     encodedData,
	}, nil
}

func (e *GasEstimator) simulateSwap(ctx context.Context, callMsg ethereum.CallMsg) (uint64, *big.Int, error) {
	result, err := e.ethClient.CallContract(ctx, callMsg, nil)
	if err != nil {
		return 0, nil, err
	}

	outputs, err := e.routerABI.Unpack("swap", result)
	if err != nil || len(outputs) != 2 {
		return 0, nil, fmt.Errorf("failed to unpack swap simulation output : %v", err)
	}

	returnAmount, ok1 := outputs[0].(*big.Int)
	if !ok1 || returnAmount == nil {
		return 0, nil, errors.New("invalid swap simulation output: returnAmount is nil")
	}

	gasUsed, ok2 := outputs[1].(*big.Int)
	if !ok2 || gasUsed == nil {
		return 0, nil, errors.New("invalid swap simulation output: gasUsed is nil")
	}

	return EstimateTotalGas(gasUsed.Uint64()), returnAmount, nil
}

// Formula derived from linear regression analysis of Tenderly API data
func EstimateTotalGas(gasUsed uint64) uint64 {
	return uint64(41329.877148 + float64(gasUsed)*1.014956)
}

func (e *GasEstimator) GetGasTokenPriceUSD(ctx context.Context) (float64, error) {
	priceByAddress, err := e.onchainpriceRepository.FindByAddresses(ctx, []string{e.gasTokenAddress})
	if err != nil {
		return 0, err
	}

	if price, ok := priceByAddress[e.gasTokenAddress]; ok && price != nil && price.USDPrice.Buy != nil {
		priceFloat, _ := price.USDPrice.Buy.Float64()
		return priceFloat, nil
	}
	return 0, nil
}

func (e *GasEstimator) getGasPrice(ctx context.Context) (*big.Float, error) {
	suggestedGasPrice, err := e.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	return new(big.Float).SetInt(suggestedGasPrice), nil
}
