package erc4626

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

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
	_ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)
	_ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)
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
		MaxDeposit:   uint256.MustFromBig(state.MaxDeposit),
		MaxRedeem:    uint256.MustFromBig(state.MaxRedeem),
		DepositRates: lo.Map(state.DepositRates, func(item *big.Int, _ int) *uint256.Int { return uint256.MustFromBig(item) }),
		RedeemRates:  lo.Map(state.RedeemRates, func(item *big.Int, _ int) *uint256.Int { return uint256.MustFromBig(item) }),
		TotalAssets:  uint256.MustFromBig(state.TotalAssets),
	})
	if err != nil {
		return errors.WithMessage(err, "json.Marshal extra")
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{lo.CoalesceOrEmpty(state.MaxRedeem, state.TotalSupply, bignumber.ZeroBI).String(),
		lo.CoalesceOrEmpty(state.MaxDeposit, state.TotalAssets, bignumber.ZeroBI).String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = state.BlockNumber
	return nil
}

func FetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, vaultAddr string, vaultCfg VaultCfg,
	fetchAsset bool, overrides map[common.Address]gethclient.OverrideAccount) (common.Address, *PoolState, error) {
	var assetToken common.Address
	poolState := PoolState{
		DepositRates: make([]*big.Int, len(PrefetchAmounts)),
		RedeemRates:  make([]*big.Int, len(PrefetchAmounts)),
	}

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	addStateCalls(func(c *ethrpc.Call, output []any) { req.AddCall(c, output) }, vaultAddr, vaultCfg, fetchAsset, &assetToken, &poolState)

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return assetToken, nil, err
	}

	normalizePoolState(&poolState)
	if resp.BlockNumber != nil {
		poolState.BlockNumber = resp.BlockNumber.Uint64()
	}
	return assetToken, &poolState, nil
}

// addStateCalls registers all RPC reads for the given vault into addFn, writing results into assetToken and state.
// Works with any caller — ethrpc.Request or LazyRequest — by accepting a generic add function.
func addStateCalls(addFn func(*ethrpc.Call, []any), vaultAddr string, vaultCfg VaultCfg, fetchAsset bool, assetToken *common.Address, state *PoolState) {
	if fetchAsset {
		addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: erc4626MethodAsset}, []any{assetToken})
	}
	if vaultCfg.Gas.Deposit > 0 {
		addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: erc4626MethodMaxDeposit, Params: []any{AddrDummy}}, []any{&state.MaxDeposit})
		addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: erc4626MethodTotalAssets}, []any{&state.TotalAssets})
		for i, amt := range PrefetchAmounts {
			addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: ERC4626MethodPreviewDeposit, Params: []any{amt.ToBig()}}, []any{&state.DepositRates[i]})
		}
	}
	if vaultCfg.Gas.Redeem > 0 {
		addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: erc4626MethodMaxRedeem, Params: []any{AddrDummy}}, []any{&state.MaxRedeem})
		addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: erc4626MethodTotalSupply}, []any{&state.TotalSupply})
		for i, amt := range PrefetchAmounts {
			addFn(&ethrpc.Call{ABI: ABI, Target: vaultAddr, Method: ERC4626MethodPreviewRedeem, Params: []any{amt.ToBig()}}, []any{&state.RedeemRates[i]})
		}
	}
}

func normalizePoolState(state *PoolState) {
	if state.MaxDeposit == nil || state.MaxDeposit.Sign() == 0 {
		state.MaxDeposit = state.TotalAssets // fallback to a sensible value
	} else if state.MaxDeposit.Cmp(bignumber.MaxUint128) > 0 {
		state.MaxDeposit = nil // no limit
	}
	if state.MaxRedeem == nil || state.MaxRedeem.Sign() == 0 {
		state.MaxRedeem = state.TotalSupply // fallback to a sensible value
	} else if state.MaxRedeem.Cmp(bignumber.MaxUint128) > 0 {
		state.MaxRedeem = nil // no limit
	}
}
