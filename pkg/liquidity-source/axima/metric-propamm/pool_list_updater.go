package metricpropamm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poolpkg.RegisterFactory0(DexType, axima.NewPoolSimulator)

type MetadataResponse struct {
	Data       []PairMetadata `json:"data"`
	Total      int            `json:"total"`
	NextOffset *int64         `json:"nextOffset"` // nil on the last page
}

type PairMetadata struct {
	Pair                    string `json:"pair"`
	PoolAddress             string `json:"poolAddress"`
	Token0                  string `json:"token0"`
	Token1                  string `json:"token1"`
	Token0Decimals          uint8  `json:"token0Decimals"`
	Token1Decimals          uint8  `json:"token1Decimals"`
	PoolFactoryAddress      string `json:"poolFactoryAddress"`
	SwapWhitelistingEnabled bool   `json:"swapWhitelistingEnabled"`
}

type PoolsListUpdater struct {
	config *axima.Config
	client *resty.Client
}

var _ = poollist.RegisterFactoryC(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *axima.Config) *PoolsListUpdater {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)
	return &PoolsListUpdater{config: config, client: client}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	pairMetadata, err := u.fetchAllPairMetadata(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools := lo.Map(pairMetadata, func(pm PairMetadata, _ int) entity.Pool {
		staticExtra, _ := json.Marshal(axima.StaticExtra{
			Pair:                    pm.Pair,
			SwapWhitelistingEnabled: pm.SwapWhitelistingEnabled,
		})

		extra, _ := json.Marshal(axima.Extra{QuoteAvailable: false, MaxAge: u.config.MaxAge, IsV2: true})
		return entity.Pool{
			Address:  strings.ToLower(pm.PoolAddress),
			Reserves: []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(pm.Token0), Decimals: pm.Token0Decimals, Swappable: true},
				{Address: strings.ToLower(pm.Token1), Decimals: pm.Token1Decimals, Swappable: true},
			},
			Exchange:    u.config.DexID,
			Type:        DexType,
			StaticExtra: string(staticExtra),
			Extra:       string(extra),
			Timestamp:   time.Now().Unix(),
		}
	})
	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) fetchAllPairMetadata(ctx context.Context) ([]PairMetadata, error) {
	var (
		all    []PairMetadata
		offset int64
	)
	for {
		var resp MetadataResponse
		res, err := u.client.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"count":  strconv.Itoa(metadataPageSize),
				"offset": strconv.FormatInt(offset, 10),
			}).
			SetResult(&resp).
			Get(fmt.Sprintf("/public/v1/evm/%d/metadata", u.config.ChainID))
		if err != nil {
			return nil, err
		} else if res.IsError() {
			return nil, fmt.Errorf("metadata API error: %s", res.String())
		}
		all = append(all, resp.Data...)

		if resp.NextOffset == nil || *resp.NextOffset <= offset || len(resp.Data) == 0 {
			return all, nil
		}
		offset = *resp.NextOffset
	}
}
