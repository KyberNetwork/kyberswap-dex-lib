package lunarbase

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func fetchRPCStateWithCache(ctx context.Context, coreAddress string, chainID valueobject.ChainID,
	client *ethrpc.Client, cache *snapshotCache) (*rpcState, error) {
	if client == nil || !common.IsHexAddress(coreAddress) || common.HexToAddress(coreAddress) == (common.Address{}) {
		return nil, fmt.Errorf("lunarbase snapshot: invalid client or pool address")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	address := common.HexToAddress(coreAddress)
	// The validating latest call must start after these state copies exist.
	// Never look up a newly inserted cache entry after the validating RPC.
	prior := cache.capture(chainID, address)
	// Even a caller-supplied canonical header can be historical. Always read
	// the current tip; BlockHeaders continues to express only a minimum.
	header, err := latestSnapshotHeader(ctx, client)
	if err != nil {
		return nil, err
	}
	ref := snapshotReference{header.Number.Uint64(), header.Hash()}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if state := prior[ref]; state != nil {
		return state, nil
	}
	state, err := fetchRPCStateAt(ctx, coreAddress, chainID, client, nil, ref)
	if err != nil {
		return nil, err
	}
	cache.put(chainID, address, state)
	return state, nil
}
