package backup

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	vaultT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/vault-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *vaultT1.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterBackupFactoryCE0(vaultT1.DexType, NewPoolTracker)

func NewPoolTracker(config *vaultT1.Config, ethrpcClient *ethrpc.Client) *PoolTracker {
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
	swapData, err := t.getPoolSwapData(ctx, p.Address, overrides)
	if swapData == nil || err != nil {
		logger.WithFields(logger.Fields{"dexType": vaultT1.DexType, "error": err}).Error("Error getPoolSwapData")
		return p, err
	}

	extra := vaultT1.PoolExtra{
		WithAbsorb: swapData.WithAbsorb,
		Ratio:      swapData.Ratio,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": vaultT1.DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{swapData.InAmt.String(), swapData.OutAmt.String()}

	return p, nil
}

func (t *PoolTracker) getPoolSwapData(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*vaultT1.SwapData, error) {
	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)

	output := &vaultT1.Swap{}
	req.AddCall(&ethrpc.Call{
		ABI:    *vaultT1.VaultLiquidationResolverABI,
		Target: t.config.VaultLiquidationResolver,
		Method: vaultT1.VLRMethodGetSwapForProtocol,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&output})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": vaultT1.DexType,
			"error":   err,
		}).Error("Error in GetSwapForProtocol Call")
		return nil, err
	}

	return &output.Data, nil
}
