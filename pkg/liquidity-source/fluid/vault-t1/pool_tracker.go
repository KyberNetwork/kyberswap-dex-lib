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

	// TODO: Get block number?
	blockNumber := uint64(20620149)

	swapData, err := t.getPoolSwapData(ctx, p.Address)
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

func (t *PoolTracker) getPoolSwapData(ctx context.Context, poolAddress string) (*SwapData, error) {
	logger.Debugf("Starting getPoolSwapData for pool: %s", poolAddress)

	req := t.ethrpcClient.R().SetContext(ctx)

	// Set the block number for the fork (test temporary) TODO REMOVE
	blockNumber := big.NewInt(20620149)
	req.SetBlockNumber(blockNumber)

	var output interface{}
	req.AddCall(&ethrpc.Call{
		ABI:    vaultLiquidationResolverABI,
		Target: vaultLiquidationResolver[t.config.ChainID],
		Method: VLRMethodGetSwapForProtocol,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&output})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Error in GetSwapForProtocol Call")
		return nil, err
	}

	castResult, ok := output.(struct {
		Path struct {
			Protocol common.Address `json:"protocol"`
			TokenIn  common.Address `json:"tokenIn"`
			TokenOut common.Address `json:"tokenOut"`
		} `json:"path"`
		Data struct {
			InAmt      *big.Int `json:"inAmt"`
			OutAmt     *big.Int `json:"outAmt"`
			WithAbsorb bool     `json:"withAbsorb"`
			Ratio      *big.Int `json:"ratio"`
		} `json:"data"`
	})
	if !ok {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Error in GetSwapForProtocol response conversion")
		return nil, err
	}

	// automatically casting to &swap instead of going via output -> castResult doesn't work
	var swap SwapData
	swap.InAmt = castResult.Data.InAmt
	swap.OutAmt = castResult.Data.OutAmt
	swap.WithAbsorb = castResult.Data.WithAbsorb
	swap.Ratio = castResult.Data.Ratio

	return &swap, nil
}
