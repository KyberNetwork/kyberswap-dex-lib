package someswapv2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       *Config
	client       *resty.Client
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(cfg.HTTPConfig.BaseURL).
		SetTimeout(cfg.HTTPConfig.Timeout.Duration).
		SetRetryCount(cfg.HTTPConfig.RetryCount)

	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		client:       client,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	apiPools, err := u.getPoolsFromAPI(ctx)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPoolsFromAPI(apiPools)
	if err != nil {
		return nil, nil, err
	}

	return pools, nil, nil
}

func (u *PoolsListUpdater) getPoolsFromAPI(ctx context.Context) ([]APIPool, error) {
	req := u.client.R().SetContext(ctx)

	var result GetPoolsResponse
	resp, err := req.SetResult(&result).Get(poolsEndpoint)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get pools from API: status %v", resp.Status())
	}

	var pools []APIPool
	for _, pair := range result.Pools {
		for _, entry := range pair.Pools {
			wToken0, err := kutils.Atou[uint32](entry.FeeConfig.WToken0In)
			if err != nil {
				return nil, err
			}
			wToken1, err := kutils.Atou[uint32](entry.FeeConfig.WToken1In)
			if err != nil {
				return nil, err
			}

			pools = append(pools, APIPool{
				PairAddress: entry.Backend.PairAddress,
				Token0:      pair.Token0,
				Token1:      pair.Token1,
				BaseFee:     entry.FeeConfig.BaseFeeBps,
				WToken0:     wToken0,
				WToken1:     wToken1,
			})
		}
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPoolsFromAPI(apiPools []APIPool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(apiPools))

	for _, ap := range apiPools {
		token0 := &entity.PoolToken{
			Address:   valueobject.ZeroToWrappedLower(ap.Token0.Address, u.config.ChainId),
			Swappable: true,
		}
		token1 := &entity.PoolToken{
			Address:   valueobject.ZeroToWrappedLower(ap.Token1.Address, u.config.ChainId),
			Swappable: true,
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			BaseFee: ap.BaseFee,
			WToken0: ap.WToken0,
			WToken1: ap.WToken1,
			Token0:  ap.Token0.Address,
			Token1:  ap.Token1.Address,
			Router:  u.config.Router,
		})
		if err != nil {
			return nil, err
		}

		newPool := entity.Pool{
			Address:     strings.ToLower(ap.PairAddress),
			SwapFee:     float64(ap.BaseFee) / float64(feeDen.Uint64()),
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{token0, token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
