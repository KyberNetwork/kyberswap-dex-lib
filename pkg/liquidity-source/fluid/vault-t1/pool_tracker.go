package vaultT1

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

type PoolTracker struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Debugf("Starting GetNewPoolState for pool: %s", p.Address)

	swapData, blockNumber, err := t.getPoolSwapData(ctx, p.Address)
	if err != nil {
		logger.Errorf("Error getting pool swap data: %v", err)
		return p, err
	}
	logger.Debugf("Retrieved swap data: %+v, blockNumber: %d", swapData, blockNumber)

	extra := struct {
		WithAbsorb bool     `json:"withAbsorb"`
		Ratio      *big.Int `json:"ratio"`
	}{
		WithAbsorb: swapData.WithAbsorb,
		Ratio:      swapData.Ratio,
	}
	logger.Debugf("Created extra struct: %+v", extra)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.Errorf("Error marshaling extra data: %v", err)
		return p, err
	}
	logger.Debugf("Marshaled extra data: %s", string(extraBytes))

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{swapData.InAmt.String(), swapData.OutAmt.String()}
	logger.Debugf("Updated pool state: %+v", p)

	return p, nil
}

func (t *PoolTracker) getPoolSwapData(ctx context.Context, poolAddress string) (*SwapData, uint64, error) {
	logger.Debugf("Starting getPoolSwapData for pool: %s", poolAddress)

	var swap Swap

	req := t.ethrpcClient.R().SetContext(ctx)
	logger.Debugf("Created ethrpc request")

	// TODO: this must be callStatic called and currently doesn't work. Once this is resolved,
	// integration can be finalized. Only pool_simulator.go must be coded which simply uses the data we
	// get here like pool reserves and the ratio / with absorb flag fed into Pool extra.
	//
	// Run test for tracker: go test -v -run TestPoolTracker ./pkg/liquidity-source/fluid/vault-t1/
	//
	req.AddCall(&ethrpc.Call{
		ABI:    vaultLiquidationResolverABI,
		Target: vaultLiquidationResolver[t.config.ChainID],
		Method: VLRMethodGetSwapForProtocol,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&swap})

	logger.Debugf("Added call to request: ABI: %v, Target: %s, Method: %s, Params: %v",
		vaultLiquidationResolverABI,
		vaultLiquidationResolver[t.config.ChainID],
		VLRMethodGetSwapForProtocol,
		common.HexToAddress(poolAddress))

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.Errorf("Error in TryBlockAndAggregate: %v", err)
		return nil, 0, err
	}
	logger.Debugf("Received response: %+v", resp)

	logger.Infof("Swap data for pool %s: %+v", poolAddress, swap.Data)

	return &swap.Data, resp.BlockNumber.Uint64(), nil
}
