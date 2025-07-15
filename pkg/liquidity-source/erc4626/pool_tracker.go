package erc4626

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger logger.Logger
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	lg := logger.WithFields(logger.Fields{
		"dexId":   cfg.DexId,
		"dexType": DexType,
	})

	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("Start updating state.")
	defer func() {
		lg.Info("Finish updating state.")
	}()

	vaultAddr := p.Tokens[0].Address
	vaultCfg := t.cfg.Vaults[vaultAddr]
	_, state, err := fetchAssetAndState(ctx, t.ethrpcClient, vaultAddr, vaultCfg, false, overrides)
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch state")

		return p, err
	}

	return p, updateEntityState(&p, vaultCfg, state)
}

func updateEntityState(p *entity.Pool, vaultCfg VaultCfg, state *PoolState) error {
	maxDeposit := uint256.MustFromBig(state.MaxDeposit)
	maxRedeem := uint256.MustFromBig(state.MaxRedeem)
	extraBytes, err := json.Marshal(Extra{
		Gas:         Gas(vaultCfg.Gas),
		SwapTypes:   vaultCfg.SwapTypes,
		MaxDeposit:  maxDeposit,
		MaxRedeem:   maxRedeem,
		DepositRate: uint256.MustFromBig(state.DepositRate),
		RedeemRate:  uint256.MustFromBig(state.RedeemRate),
	})
	if err != nil {
		return errors.WithMessage(err, "json.Marshal extra")
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{safeToString(maxRedeem, "0"), safeToString(maxDeposit, "0")}
	p.Extra = string(extraBytes)
	p.BlockNumber = state.blockNumber
	return nil
}

func safeToString(value *uint256.Int, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return value.String()
}

func fetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, vaultAddr string, vaultCfg VaultCfg,
	fetchAsset bool, overrides map[common.Address]gethclient.OverrideAccount) (common.Address, *PoolState, error) {
	var assetToken common.Address
	var poolState PoolState

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	if fetchAsset {
		req = req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodAsset,
		}, []any{&assetToken})
	}

	if vaultCfg.SwapTypes == Both || vaultCfg.SwapTypes == Deposit {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxDeposit,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodPreviewDeposit,
			Params: []any{BiWad},
		}, []any{&poolState.DepositRate})
	}
	if vaultCfg.SwapTypes == Both || vaultCfg.SwapTypes == Redeem {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxRedeem,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxRedeem}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodPreviewRedeem,
			Params: []any{BiWad},
		}, []any{&poolState.RedeemRate})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return assetToken, nil, err
	}

	if poolState.DepositRate != nil && poolState.DepositRate.Sign() <= 0 {
		poolState.DepositRate = nil
	}
	if poolState.RedeemRate != nil && poolState.RedeemRate.Sign() <= 0 {
		poolState.RedeemRate = nil
	}
	if poolState.MaxDeposit != nil && poolState.MaxDeposit.Cmp(bignumber.MAX_UINT_256) == 0 {
		poolState.MaxDeposit = nil
	}
	if poolState.MaxRedeem != nil && (poolState.MaxRedeem.Sign() == 0 || poolState.MaxRedeem.Cmp(bignumber.MAX_UINT_256) == 0) {
		poolState.MaxRedeem = nil
	}

	if resp.BlockNumber != nil {
		poolState.blockNumber = resp.BlockNumber.Uint64()
	}
	return assetToken, &poolState, nil
}
