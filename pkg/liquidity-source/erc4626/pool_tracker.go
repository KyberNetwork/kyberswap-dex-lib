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

var (
	_ = pooltrack.RegisterBackupFactoryCE0(DexType, NewPoolTracker)
)

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
	vaultCfg, ok := t.cfg.Vaults[vaultAddr]
	if !ok { // manually added vault
		var extra Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		vaultCfg.Gas = GasCfg(extra.Gas)
	}
	_, state, err := FetchAssetAndState(ctx, t.ethrpcClient, vaultAddr, vaultCfg, false, overrides)
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch state")

		return p, err
	}

	return p, UpdateEntityState(&p, vaultCfg, state)
}

func UpdateEntityState(p *entity.Pool, vaultCfg VaultCfg, state *PoolState) error {
	extraBytes, err := json.Marshal(Extra{
		Gas:          Gas(vaultCfg.Gas),
		MaxDeposit:   state.MaxDeposit,
		MaxRedeem:    state.MaxRedeem,
		DepositRates: state.DepositRates,
		RedeemRates:  state.RedeemRates,
		TotalAssets:  state.TotalAssets,
	})
	if err != nil {
		return errors.WithMessage(err, "json.Marshal extra")
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		coalesceUint256Dec(state.MaxDeposit, state.TotalAssets),
		coalesceUint256Dec(state.MaxRedeem, state.TotalSupply),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = state.BlockNumber
	return nil
}

func FetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, vaultAddr string, vaultCfg VaultCfg,
	fetchAsset bool, overrides map[common.Address]gethclient.OverrideAccount) (common.Address, *PoolState, error) {
	var (
		assetToken common.Address
		poolState  = PoolState{
			DepositRates: make([]*uint256.Int, len(PrefetchAmounts)),
			RedeemRates:  make([]*uint256.Int, len(PrefetchAmounts)),
		}
	)

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	if fetchAsset {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: Erc4626MethodAsset,
		}, []any{&assetToken})
	}

	if vaultCfg.Gas.Deposit > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: Erc4626MethodMaxDeposit,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: Erc4626MethodTotalAssets,
		}, []any{&poolState.TotalAssets})

		for i, amt := range PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    ABI,
				Target: vaultAddr,
				Method: Erc4626MethodPreviewDeposit,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.DepositRates[i]})
		}
	}
	if vaultCfg.Gas.Redeem > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: Erc4626MethodMaxRedeem,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxRedeem}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: Erc4626MethodTotalSupply,
		}, []any{&poolState.TotalSupply})

		for i, amt := range PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    ABI,
				Target: vaultAddr,
				Method: Erc4626MethodPreviewRedeem,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.RedeemRates[i]})
		}
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return assetToken, nil, err
	}

	if poolState.MaxDeposit == nil || poolState.MaxDeposit.IsZero() {
		poolState.MaxDeposit = poolState.TotalAssets // fallback to a sensible value
	} else if poolState.MaxDeposit.ToBig().Cmp(bignumber.MaxUint128) > 0 {
		poolState.MaxDeposit = nil // no limit
	}
	if poolState.MaxRedeem == nil || poolState.MaxRedeem.IsZero() {
		poolState.MaxRedeem = poolState.TotalSupply // fallback to a sensible value
	} else if poolState.MaxRedeem.ToBig().Cmp(bignumber.MaxUint128) > 0 {
		poolState.MaxRedeem = nil // no limit
	}

	if resp.BlockNumber != nil {
		poolState.BlockNumber = resp.BlockNumber.Uint64()
	}
	return assetToken, &poolState, nil
}

func coalesceUint256Dec(primary, fallback *uint256.Int) string {
	if primary != nil {
		return primary.Dec()
	}
	if fallback != nil {
		return fallback.Dec()
	}
	return "0"
}
