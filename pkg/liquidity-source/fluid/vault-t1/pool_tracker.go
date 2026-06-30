package vaultT1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	d := &rpcData{output: &Swap{}}
	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, t.config.VaultLiquidationResolver, d)

	if _, err := req.Call(); err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error in GetSwapForProtocol Call")
		return p, err
	}

	if d.output == nil {
		logger.WithFields(logger.Fields{"dexType": DexType}).Error("Error getPoolSwapData")
		return p, nil
	}

	return buildPoolState(p, d, nil)
}

type rpcData struct {
	output *Swap
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress, resolverAddress string, d *rpcData) {
	addFn(&ethrpc.Call{
		ABI:    vaultLiquidationResolverABI,
		Target: resolverAddress,
		Method: VLRMethodGetSwapForProtocol,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&d.output})
}

func buildPoolState(p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	swapData := &d.output.Data

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
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{swapData.InAmt.String(), swapData.OutAmt.String()}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	return p, nil
}
