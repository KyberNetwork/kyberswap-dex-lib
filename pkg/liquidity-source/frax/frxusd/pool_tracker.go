package frxusd

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{
		"address": p.Address,
	})
	lg.Info("start updating state")
	defer func() {
		lg.Info("finish updating state")
	}()

	vaultCfg, ok := t.cfg.Vaults[p.Address]
	if !ok {
		lg.Error("vault config not found")
		return p, nil
	}

	state, err := FetchAssetAndState(ctx, t.ethrpcClient, p.Address, vaultCfg, nil)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Errorf("failed to fetch state")

		return p, err
	}

	return p, erc4626.UpdateEntityState(&p, vaultCfg, state)
}

func FetchAssetAndState(ctx context.Context, ethrpcClient *ethrpc.Client, vaultAddr string, vaultCfg erc4626.VaultCfg,
	overrides map[common.Address]gethclient.OverrideAccount) (*erc4626.PoolState, error) {
	var (
		poolState = erc4626.PoolState{
			DepositRates: make([]*big.Int, len(erc4626.PrefetchAmounts)),
			RedeemRates:  make([]*big.Int, len(erc4626.PrefetchAmounts)),
		}
		mdwrCombo struct {
			MaxAssetsDepositable  *big.Int
			MaxSharesMintable     *big.Int
			MaxAssetsWithdrawable *big.Int
			MaxSharesRedeemable   *big.Int
		}
	)

	req := ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{
		ABI:    FrxUsdCustodianUsdcABI,
		Target: vaultAddr,
		Method: "mdwrComboView",
	}, []any{&mdwrCombo})

	if vaultCfg.SwapTypes == erc4626.Both || vaultCfg.SwapTypes == erc4626.Deposit {
		for i, amt := range erc4626.PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    erc4626.ABI,
				Target: vaultAddr,
				Method: erc4626.ERC4626MethodPreviewDeposit,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.DepositRates[i]})
		}
	}

	if vaultCfg.SwapTypes == erc4626.Both || vaultCfg.SwapTypes == erc4626.Redeem {
		for i, amt := range erc4626.PrefetchAmounts {
			req.AddCall(&ethrpc.Call{
				ABI:    erc4626.ABI,
				Target: vaultAddr,
				Method: erc4626.ERC4626MethodPreviewRedeem,
				Params: []any{amt.ToBig()},
			}, []any{&poolState.RedeemRates[i]})
		}
	}

	resp, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	poolState.MaxDeposit = mdwrCombo.MaxAssetsDepositable
	poolState.MaxRedeem = mdwrCombo.MaxSharesRedeemable
	poolState.TotalAssets = mdwrCombo.MaxAssetsWithdrawable
	poolState.TotalSupply = mdwrCombo.MaxSharesRedeemable

	if resp.BlockNumber != nil {
		poolState.BlockNumber = resp.BlockNumber.Uint64()
	}

	return &poolState, nil
}
