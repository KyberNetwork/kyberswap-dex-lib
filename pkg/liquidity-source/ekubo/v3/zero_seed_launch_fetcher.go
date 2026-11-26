package ekubov3

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

func (f *dataFetchers) fetchLaunchPools(ctx context.Context, keys []pools.AnyPoolKey, overrides map[common.Address]gethclient.OverrideAccount) ([]fetchedPool, error) {
	header, err := f.ethrpcClient.GetETHClient().HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("reading launch snapshot header: %w", err)
	}
	result := make([]fetchedPool, 0, len(keys))
	for _, batch := range lo.Chunk(keys, maxBatchSize) {
		req := f.ethrpcClient.R().SetContext(ctx).SetBlockNumber(header.Number)
		if overrides != nil {
			req.SetOverrides(overrides)
		}
		data := make([]quoteData, len(batch))
		launches := make([]struct{ Launch pools.Launch }, len(batch))
		abiKeys := make([]pools.AbiPoolKey, len(batch))
		for i, key := range batch {
			abiKeys[i] = key.ToAbi()
			id, err := key.NumId()
			if err != nil {
				return nil, err
			}
			req.AddCall(&ethrpc.Call{ABI: abis.ZeroSeedLaunchABI, Target: key.Extension().Hex(), Method: "getLaunch", Params: []any{common.BytesToHash(id)}}, []any{&launches[i]})
		}
		// The derived launch range spans at most 256 spacings. Fetch both
		// installed bounds, including when the active price is at either edge.
		req.AddCall(&ethrpc.Call{ABI: abis.QuoteDataFetcherABI, Target: f.config.QuoteDataFetcher, Method: quoteDataFetcherMethod, Params: []any{abiKeys, uint32(1)}}, []any{&data})
		if _, err := req.Aggregate(); err != nil {
			return nil, fmt.Errorf("reading launch snapshot: %w", err)
		}
		for i, key := range batch {
			config, ok := key.Config.TypeConfig.(pools.ConcentratedPoolTypeConfig)
			if !ok {
				return nil, fmt.Errorf("launch requires a concentrated pool")
			}
			launch, err := pools.NewZeroSeedLaunchPool(key.ToConcentrated(config), &pools.ZeroSeedLaunchPoolState{
				UnderlyingPoolState: newConcentratedPoolState(&data[i]), Launch: launches[i].Launch, BlockTimestamp: header.Time,
			})
			if err != nil {
				return nil, err
			}
			result = append(result, fetchedPool{PoolWithBlockNumber: PoolWithBlockNumber{Pool: launch, blockNumber: header.Number.Uint64()}, key: key})
		}
	}
	// Do not combine pre-reorg headers with post-reorg state at the same height.
	canonical, err := f.ethrpcClient.GetETHClient().HeaderByNumber(ctx, header.Number)
	if err != nil {
		return nil, err
	}
	if canonical.Hash() != header.Hash() {
		return nil, ErrReorg
	}
	return result, nil
}
