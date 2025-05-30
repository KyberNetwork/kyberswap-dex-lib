package iziswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	iziswapclient "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/client"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// This integration is mostly forked from https://github.com/opcc22059/kyberswap-dex-lib/tree/iZiSwap,
// with minor changes in PoolsListUpdater and PoolSimulator.

type PoolsListUpdater struct {
	config *Config
	client IClient
}

var _ = poollist.RegisterFactoryC(DexTypeiZiSwap, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config) *PoolsListUpdater {
	httpClient := iziswapclient.NewHTTPClient(&cfg.HTTP)
	return &PoolsListUpdater{
		config: cfg,
		client: httpClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAtTimestamp: 0,
	}

	if metadataBytes != nil || len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	params := ListPoolsParams{
		ChainId:   d.config.ChainID,
		Version:   "v2",
		TimeStart: metadata.LastCreatedAtTimestamp,
		Limit:     d.config.NewPoolLimit,
	}

	queryResult, err := d.client.ListPools(ctx, params)
	logger.Infof("got %v pools from iZiSwap API", len(queryResult))

	if err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(queryResult))
	latestTimestamp := metadata.LastCreatedAtTimestamp

	for _, p := range queryResult {
		if p.TokenXAddress == "" || p.TokenYAddress == "" {
			continue
		}
		tokens := make([]*entity.PoolToken, 0, 2)
		reserves := make([]string, 0, 2)

		token0Model := entity.PoolToken{
			Address:   p.TokenXAddress,
			Symbol:    p.TokenX,
			Decimals:  uint8(p.TokenXDecimals),
			Swappable: true,
		}
		tokens = append(tokens, &token0Model)
		reserves = append(reserves, "0")

		token1Model := entity.PoolToken{
			Address:   p.TokenYAddress,
			Symbol:    p.TokenY,
			Decimals:  uint8(p.TokenYDecimals),
			Swappable: true,
		}
		tokens = append(tokens, &token1Model)
		reserves = append(reserves, "0")

		var swapFee = float64(p.Fee)
		var newPool = entity.Pool{
			Address:      p.Address,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      swapFee,
			Exchange:     d.config.DexID,
			Type:         DexTypeiZiSwap,
			Timestamp:    time.Now().Unix(),
			Reserves:     reserves,
			Tokens:       tokens,
		}
		pools = append(pools, newPool)
		if latestTimestamp < p.Timestamp {
			latestTimestamp = p.Timestamp
		}
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAtTimestamp: latestTimestamp,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}
