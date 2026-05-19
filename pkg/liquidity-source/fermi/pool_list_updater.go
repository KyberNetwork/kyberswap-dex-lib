package fermi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type pairInfo struct {
	TokenIn  common.Address
	TokenOut common.Address
	IsActive bool
}

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{"dex_id": u.config.DexId}).Info("started getting new pools")

	pairs, err := u.fetchPairs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("fetchPairs: %w", err)
	}

	pools := make([]entity.Pool, 0, len(pairs))
	for _, pair := range pairs {
		if !pair.IsActive {
			continue
		}
		p, err := u.buildPool(pair)
		if err != nil {
			return nil, nil, fmt.Errorf("buildPool %s/%s: %w", pair.TokenIn.Hex(), pair.TokenOut.Hex(), err)
		}
		pools = append(pools, p)
	}

	logger.WithFields(logger.Fields{
		"dex_id":     u.config.DexId,
		"pool_count": len(pools),
	}).Info("finished getting new pools")

	return pools, nil, nil
}

func (u *PoolsListUpdater) fetchPairs(ctx context.Context) ([]pairInfo, error) {
	var pairs []pairInfo
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    fermiSwapperABI,
		Target: u.config.FermiSwapper,
		Method: methodGetPairs,
	}, []any{&pairs})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	return pairs, nil
}

func (u *PoolsListUpdater) buildPool(pair pairInfo) (entity.Pool, error) {
	t0 := strings.ToLower(pair.TokenIn.Hex())
	t1 := strings.ToLower(pair.TokenOut.Hex())

	swapper := strings.ToLower(u.config.FermiSwapper)
	poolAddr := fmt.Sprintf("%s_%s_%s", swapper, t0, t1)

	staticExtraBytes, err := json.Marshal(StaticExtra{FermiSwapper: swapper})
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     poolAddr,
		Exchange:    u.config.DexId,
		Type:        DexType,
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: t0, Swappable: true},
			{Address: t1, Swappable: true},
		},
		Timestamp: time.Now().Unix(),
	}, nil
}
