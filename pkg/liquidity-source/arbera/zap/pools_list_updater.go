package arberazap

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	// - 0xbaadcc2962417c01af99fb2b7c75706b9bd6babe        # LBGT
	// - 0xface73a169e2ca2934036c8af9f464b5de9ef0ca        # stLBGT 	erc4626
	// - 0x883899d0111d69f85fdfd19e4b89e613f231b781        # brLBGT 	den
	// - 0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7        # brARBERO den
	// - 0xfa7767bbb3d832217abaa86e5f2654429b3bf29f        # ARBERO
	// - 0x3fd02eaddb07080b8e2640afb6d52f10d6396926        # stARBERO
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.initialized {
		logger.Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	var errs []error
	pools := make([]entity.Pool, 0, len(u.cfg.Pools))
	for poolAddr, poolCfg := range u.cfg.Pools {
		pool, err := u.getNewPool(ctx, poolAddr, poolCfg)
		if err != nil {
			errs = append(errs, errors.WithMessage(err, poolAddr))
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

func (u *PoolsListUpdater) getNewPool(_ context.Context, poolAddr string, poolCfg PoolCfg) (*entity.Pool, error) {
	staticExtra, err := json.Marshal(StaticExtra{
		BasePools: []string{poolCfg.VaultToken, poolCfg.Den1Token, poolCfg.DenAmmPool, poolCfg.Den2Token, poolCfg.LstToken},
	})
	if err != nil {
		return nil, err
	}

	tokens := []string{
		poolCfg.LeftToken, poolCfg.VaultToken, poolCfg.Den1Token, poolCfg.Den2Token, poolCfg.RightToken, poolCfg.LstToken,
	}
	p := &entity.Pool{
		Address:  strings.ToLower(poolAddr),
		Exchange: u.cfg.DexID,
		Type:     DexType,
		Tokens: lo.Map(tokens, func(token string, _ int) *entity.PoolToken {
			return &entity.PoolToken{
				Address:   strings.ToLower(token),
				Swappable: true,
			}
		}),
		Reserves:    lo.Map(tokens, func(token string, _ int) string { return defaultReserve }),
		Extra:       "{}",
		StaticExtra: string(staticExtra),
	}
	return p, nil
}
