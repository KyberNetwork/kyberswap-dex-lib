package erc7575

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	erc4626lazy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626/lazy"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger      logger.Logger
	initialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       logger.WithFields(logger.Fields{"dexId": cfg.DexId, "dexType": DexType}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.initialized {
		return nil, metadataBytes, nil
	}

	var errs []error
	pools := make([]entity.Pool, 0, len(u.cfg.Vaults))
	for vaultAddr, vaultCfg := range u.cfg.Vaults {
		pool, err := u.getNewPool(ctx, vaultAddr, vaultCfg)
		if err != nil {
			errs = append(errs, errors.WithMessage(err, vaultAddr))
		} else {
			pools = append(pools, *pool)
		}
	}

	if len(errs) > 0 {
		return nil, metadataBytes, errors.Errorf("failed to get new pools: %v", errs)
	}

	u.initialized = true
	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getNewPool(ctx context.Context, vaultAddr string, vaultCfg erc4626.VaultCfg) (*entity.Pool, error) {
	// Discover the (decoupled) share token via share(). This is ERC7575-specific and owned here, not in erc4626.
	shareToken := u.fetchShare(ctx, vaultAddr)

	// Reuse erc4626 for asset + state. tokenAddr is empty here (Tokens[0] isn't persisted yet), so the initial
	// totalSupply targets the vault - a harmless placeholder at discovery; the tracker reads it from the share
	// token on the next refresh. All logic getters target the vault entrypoint.
	assetToken, state, err := erc4626lazy.FetchAssetAndState(ctx, u.ethrpcClient, vaultAddr, "", vaultCfg, true, nil)
	if err != nil {
		u.logger.WithFields(logger.Fields{"error": err}).Error("failed to FetchAssetAndState")
		return nil, err
	}

	// Fall back to the vault address if share() reverts (behaves like a standard self-share vault).
	shareTokenAddr := strings.ToLower(vaultAddr)
	if shareToken != (common.Address{}) {
		shareTokenAddr = hexutil.Encode(shareToken[:])
	}

	staticExtraBytes, err := json.Marshal(erc4626.StaticExtra{
		IsNativeAsset: valueobject.IsNative(hexutil.Encode(assetToken[:])),
	})
	if err != nil {
		return nil, err
	}

	p := &entity.Pool{
		Address:  strings.ToLower(vaultAddr),
		Exchange: u.cfg.DexId,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: shareTokenAddr, Swappable: true},
			{Address: valueobject.WrapNativeLower(hexutil.Encode(assetToken[:]), u.cfg.ChainId), Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
	}

	return p, erc4626.UpdateEntityState(p, vaultCfg, state)
}

// fetchShare reads the vault's share() getter. Uses a tolerant multicall so a revert (a self-share vault that
// doesn't implement share()) leaves the zero address, letting the caller fall back to the vault address.
func (u *PoolsListUpdater) fetchShare(ctx context.Context, vaultAddr string) common.Address {
	var share common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: erc7575ABI, Target: vaultAddr, Method: methodShare}, []any{&share}).
		TryBlockAndAggregate(); err != nil {
		u.logger.WithFields(logger.Fields{"error": err, "vault": vaultAddr}).Warn("failed to fetch share()")
	}
	return share
}
