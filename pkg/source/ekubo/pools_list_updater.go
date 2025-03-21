package ekubo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type allPoolsJson = []struct {
	CoreAddress addressWrapper `json:"core_address"`
	Token0      addressWrapper `json:"token0"`
	Token1      addressWrapper `json:"token1"`
	Fee         uint64Wrapper  `json:"fee"`
	TickSpacing uint32         `json:"tick_spacing"`
	Extension   addressWrapper `json:"extension"`
}

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	registeredPools map[string]bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,

		registeredPools: make(map[string]bool),
	}
}

func (d *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]quoting.PoolKey, error) {
	poolsUrl, err := url.JoinPath(d.config.ApiUrl, "v1/poolKeys")
	if err != nil {
		return nil, fmt.Errorf("URL creation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		poolsUrl,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unknown HTTP status %s", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response failed: %w", err)
	}

	var allPools allPoolsJson
	err = json.Unmarshal(bytes, &allPools)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling JSON: %w", err)
	}

	newPoolKeys := make([]quoting.PoolKey, 0)

	for _, pool := range allPools {
		if pool.CoreAddress.Cmp(common.HexToAddress(d.config.Core)) != 0 {
			continue
		}

		poolKey := quoting.PoolKey{
			Token0: pool.Token0.Address,
			Token1: pool.Token1.Address,
			Config: quoting.Config{
				Fee:         pool.Fee.uint64,
				TickSpacing: pool.TickSpacing,
				Extension:   pool.Extension.Address,
			},
		}

		if d.registeredPools[poolKey.StringId()] {
			continue
		}

		newPoolKeys = append(newPoolKeys, poolKey)
	}

	return newPoolKeys, nil
}

func (d *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	newPoolKeys, err := d.getNewPoolKeys(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get new pool keys: %w", err)
	}

	newPools, err := fetchPools(
		ctx,
		d.ethrpcClient,
		d.config.DataFetcher,
		newPoolKeys,
		d.config.Extensions,
		d.registeredPools,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching pool states: %w", err)
	}

	return newPools, nil, nil
}
