package uniswapv3

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// FetchTickSpacings fetches tick spacings of pools with given poolAddresses.
// Return a map between poolAddress to its tick spacing.
func FetchTickSpacings(
	ctx context.Context,
	poolAddresses []string,
	ethrpcClient *ethrpc.Client,
	poolABI abi.ABI,
	methodTickSpacing string,
) (map[string]uint64, error) {
	tickSpacings := make(map[string]uint64, len(poolAddresses))

	for i := 0; i < len(poolAddresses); i += rpcChunkSize {
		endIndex := min(i+rpcChunkSize, len(poolAddresses))

		chunk := poolAddresses[i:endIndex]

		rpcRequest := ethrpcClient.NewRequest().SetContext(ctx)
		rpcResponse := make([]*big.Int, len(chunk))

		for j, poolAddress := range chunk {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: poolAddress,
				Method: methodTickSpacing,
				Params: nil,
			}, []interface{}{&rpcResponse[j]})
		}

		_, err := rpcRequest.TryAggregate()
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Error("[fetchTickSpacings] failed to process tryAggregate")
			return nil, err
		}

		for j := 0; j < len(chunk); j++ {
			poolAddress := poolAddresses[i+j]
			tickSpacings[poolAddress] = rpcResponse[j].Uint64()
		}
	}

	return tickSpacings, nil
}
