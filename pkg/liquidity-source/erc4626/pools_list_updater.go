package erc4626

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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
		"pool":    strings.ToLower(cfg.Vault),
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

	state, err := fetchState(ctx, u.ethrpcClient, u.cfg, nil)
	if err != nil {
		u.logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to fetch state")

		return nil, nil, err
	}

	u.initialized = true

	return []entity.Pool{{
		Address:  strings.ToLower(u.cfg.Vault),
		Exchange: u.cfg.DexId,
		Type:     DexType,
		Reserves: entity.PoolReserves{state.TotalSupply.String(), state.TotalAssets.String()},
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(u.cfg.ShareToken), Swappable: true},
			{Address: strings.ToLower(u.cfg.AssetToken), Swappable: true},
		},
		Timestamp:   time.Now().Unix(),
		BlockNumber: state.blockNumber,
	}}, metadataBytes, nil
}

func fetchState(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	cfg *Config,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolState, error) {
	req := ethrpcClient.NewRequest().SetContext(ctx)

	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var poolState PoolState

	req.AddCall(&ethrpc.Call{
		ABI:    ABI,
		Target: cfg.Vault,
		Method: erc4626MethodTotalSupply,
	}, []any{&poolState.TotalSupply}).AddCall(&ethrpc.Call{
		ABI:    ABI,
		Target: cfg.Vault,
		Method: erc4626MethodTotalAssets,
	}, []any{&poolState.TotalAssets})

	if cfg.SwapTypes == Both || cfg.SwapTypes == Deposit {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: cfg.Vault,
			Method: erc4626MethodMaxDeposit,
			Params: []any{eth.AddressZero},
		}, []any{&poolState.MaxDeposit})
	}
	if cfg.SwapTypes == Both || cfg.SwapTypes == Redeem {
		req.AddCall(&ethrpc.Call{
			ABI:    ABI,
			Target: cfg.Vault,
			Method: erc4626MethodMaxRedeem,
			Params: []any{eth.AddressZero},
		}, []any{&poolState.MaxRedeem})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	if resp.BlockNumber != nil {
		poolState.blockNumber = resp.BlockNumber.Uint64()
	}

	return &poolState, nil
}
