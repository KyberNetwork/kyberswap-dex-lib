package lazy

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
	erc4626 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *erc4626.Config
	ethrpcClient *ethrpc.Client

	logger logger.Logger
}

var (
	_ poolpkg.IBatchRPCPoolTracker = (*PoolTracker)(nil)
	_                              = pooltrack.RegisterFactoryCE0(erc4626.DexType, NewPoolTracker)
)

func NewPoolTracker(cfg *erc4626.Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	lg := logger.WithFields(logger.Fields{
		"dexId":   cfg.DexId,
		"dexType": erc4626.DexType,
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
		var extra erc4626.Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		vaultCfg.Gas = erc4626.GasCfg(extra.Gas)
	}
	// standard ERC4626: vault == share, so tokenAddr is empty and totalSupply targets the vault
	_, state, err := FetchAssetAndState(ctx, t.ethrpcClient, vaultAddr, "", vaultCfg, false, overrides)
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch state")

		return p, err
	}

	return p, UpdateEntityState(&p, vaultCfg, state)
}

func UpdateEntityState(p *entity.Pool, vaultCfg erc4626.VaultCfg, state *erc4626.PoolState) error {
	extraBytes, err := json.Marshal(erc4626.Extra{
		Gas:          erc4626.Gas(vaultCfg.Gas),
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

// FetchAssetAndState fetches the vault state. vaultAddr is the entrypoint all logic getters target; tokenAddr
// is the share ERC20 whose totalSupply is read (empty => vault, i.e. standard ERC4626).
func FetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, vaultAddr, tokenAddr string, vaultCfg erc4626.VaultCfg,
	fetchAsset bool, overrides map[common.Address]gethclient.OverrideAccount) (common.Address, *erc4626.PoolState, error) {
	var assetToken common.Address
	poolState := erc4626.PoolState{
		DepositRates: make([]*big.Int, len(erc4626.PrefetchAmounts)),
		RedeemRates:  make([]*big.Int, len(erc4626.PrefetchAmounts)),
	}

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	addStateCalls(func(c *ethrpc.Call, output []any) { req.AddCall(c, output) }, vaultAddr, tokenAddr, vaultCfg, fetchAsset, &assetToken, &poolState)

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

// addStateCalls registers all RPC reads for the given vault into addFn, writing results into assetToken and
// state. Works with any caller — ethrpc.Request or LazyRequest — by accepting a generic add function.
// All vault-logic getters are called on vaultAddr (the vault entrypoint). totalSupply belongs to the share
// ERC20, so it is read from tokenAddr; for a standard ERC4626 the share is the vault, and callers pass an
// empty tokenAddr to target the vault. Decoupled integrations (e.g. erc7575) pass the share token address.
func addStateCalls(addFn func(*ethrpc.Call, []any), vaultAddr, tokenAddr string, vaultCfg erc4626.VaultCfg, fetchAsset bool, assetToken *common.Address, state *erc4626.PoolState) {
	totalSupplyTarget := vaultAddr
	if tokenAddr != "" {
		totalSupplyTarget = tokenAddr
	}

	if fetchAsset {
		addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodAsset}, []any{assetToken})
	}
	if vaultCfg.Gas.Deposit > 0 {
		addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodMaxDeposit, Params: []any{erc4626.AddrDummy}}, []any{&state.MaxDeposit})
		addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodTotalAssets}, []any{&state.TotalAssets})
		for i, amt := range erc4626.PrefetchAmounts {
			addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodPreviewDeposit, Params: []any{amt.ToBig()}}, []any{&state.DepositRates[i]})
		}
	}
	if vaultCfg.Gas.Redeem > 0 {
		addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodMaxRedeem, Params: []any{erc4626.AddrDummy}}, []any{&state.MaxRedeem})
		addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: totalSupplyTarget, Method: erc4626.Erc4626MethodTotalSupply}, []any{&state.TotalSupply})
		for i, amt := range erc4626.PrefetchAmounts {
			addFn(&ethrpc.Call{ABI: erc4626.ABI, Target: vaultAddr, Method: erc4626.Erc4626MethodPreviewRedeem, Params: []any{amt.ToBig()}}, []any{&state.RedeemRates[i]})
		}
	}
}

func normalizePoolState(state *erc4626.PoolState) {
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
