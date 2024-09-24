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
	swapData, blockNumber, err := t.getPoolSwapData(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error getPoolSwapData")
		return p, err
	}

	extra := PoolExtra{
		WithAbsorb: swapData.WithAbsorb,
		Ratio:      swapData.Ratio,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{swapData.InAmt.String(), swapData.OutAmt.String()}

	return p, nil
}

func (t *PoolTracker) getPoolSwapData(ctx context.Context, poolAddress string) (*SwapData, uint64, error) {
	req := t.ethrpcClient.R().SetContext(ctx)

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get block number")
		return nil, 0, err
	}
	req.SetBlockNumber(big.NewInt(int64(blockNumber)))

	output := &Swap{}
	req.AddCall(&ethrpc.Call{
		ABI:    vaultLiquidationResolverABI,
		Target: vaultLiquidationResolver[t.config.ChainID],
		Method: VLRMethodGetSwapForProtocol,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&output})

	_, err = req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Error in GetSwapForProtocol Call")
		return nil, 0, err
	}

	return &output.Data, blockNumber, nil
}
