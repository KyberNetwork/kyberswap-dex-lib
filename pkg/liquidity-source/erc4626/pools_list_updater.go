package erc4626

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	logger      logger.Logger
	initialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	lg := logger.WithFields(logger.Fields{
		"dexId":   cfg.DexId,
		"dexType": DexType,
	})

	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list.")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	if u.initialized {
		u.logger.Infof("Pools have been initialized.")
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

func (u *PoolsListUpdater) getNewPool(ctx context.Context, vaultAddr string, vaultCfg VaultCfg) (*entity.Pool, error) {
	assetToken, state, err := fetchAssetAndState(ctx, u.ethrpcClient, vaultAddr, vaultCfg, true, nil)
	if err != nil {
		u.logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetchAssetAndState")
		return nil, err
	}

	extraBytes, _ := json.Marshal(Extra{
		Gas:         Gas(vaultCfg.Gas),
		SwapTypes:   vaultCfg.SwapTypes,
		MaxDeposit:  uint256.MustFromBig(state.MaxDeposit),
		MaxRedeem:   uint256.MustFromBig(state.MaxRedeem),
		EntryFeeBps: state.EntryFeeBps,
		ExitFeeBps:  state.ExitFeeBps,
	})

	return &entity.Pool{
		Address:   strings.ToLower(vaultAddr),
		Exchange:  u.cfg.DexId,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{state.TotalSupply.String(), state.TotalAssets.String()},
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(vaultAddr), Swappable: true},
			{Address: hexutil.Encode(assetToken[:]), Swappable: true},
		},
		Extra:       string(extraBytes),
		BlockNumber: state.blockNumber,
	}, nil
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
	req = req.AddCall(&ethrpc.Call{
		ABI:    ABI,
		Target: vaultAddr,
		Method: erc4626MethodTotalSupply,
	}, []any{&poolState.TotalSupply}).AddCall(&ethrpc.Call{
		ABI:    ABI,
		Target: vaultAddr,
		Method: erc4626MethodTotalAssets,
	}, []any{&poolState.TotalAssets})

	if vaultCfg.SwapTypes == Both || vaultCfg.SwapTypes == Deposit {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxDeposit,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodEntryFeeBasisPoints,
		}, []any{&poolState.EntryFeeBps})
	}
	var minRedeemRatio uint64
	if vaultCfg.SwapTypes == Both || vaultCfg.SwapTypes == Redeem {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMaxRedeem,
			Params: []any{AddrDummy},
		}, []any{&poolState.MaxRedeem}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodExitFeeBasisPoints,
		}, []any{&poolState.ExitFeeBps}).AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: vaultAddr,
			Method: erc4626MethodMinRedeemRatio,
		}, []any{&minRedeemRatio})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return assetToken, nil, err
	}

	if poolState.MaxDeposit != nil && poolState.MaxDeposit.Cmp(bignumber.MAX_UINT_256) == 0 {
		poolState.MaxDeposit = nil
	}
	if poolState.MaxRedeem != nil && (poolState.MaxRedeem.Sign() == 0 || poolState.MaxRedeem.Cmp(bignumber.MAX_UINT_256) == 0) {
		poolState.MaxRedeem = nil
	}
	if minRedeemRatio > 0 {
		poolState.ExitFeeBps = uint64(math.Floor(
			Bps - (Bps-float64(poolState.ExitFeeBps))*float64(minRedeemRatio)/RatioPrecision))
	}

	if resp.BlockNumber != nil {
		poolState.blockNumber = resp.BlockNumber.Uint64()
	}
	return assetToken, &poolState, nil
}
