package poolpublisher

import (
	"context"
	"fmt"
	"sync"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/aevm/types"
	dexlibencode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
)

type (
	preparedPoolIDs = map[string]struct{}
	preparedPools   = map[string]poolpkg.IPoolSimulator
)

type PoolsPublisher struct {
	aevmClient            aevmclient.Client
	publishedPoolIDs      sync.Map // map[string]preparedPoolIDs
	publishedPools        sync.Map // map[string]preparedPools
	storageIDs            []string
	storageIDsMu          sync.Mutex
	numStorageIDsToRetain int
}

func NewPoolPublisher(aevmClient aevmclient.Client, numStates int) (*PoolsPublisher, error) {
	return &PoolsPublisher{
		aevmClient:            aevmClient,
		numStorageIDsToRetain: numStates,
	}, nil
}

func (p *PoolsPublisher) PublishedPoolIDs(storageID string) map[string]struct{} {
	if addrs, ok := p.publishedPoolIDs.Load(storageID); ok {
		return addrs.(preparedPoolIDs)
	}
	return nil
}

func (p *PoolsPublisher) PublishedPools(storageID string) map[string]poolpkg.IPoolSimulator {
	if pools, ok := p.publishedPools.Load(storageID); ok {
		return pools.(preparedPools)
	}
	return nil
}

func (p *PoolsPublisher) Publish(ctx context.Context, pools map[string]poolpkg.IPoolSimulator) (string, error) {
	start := time.Now()
	encoded, err := dexlibencode.EncodePoolSimulatorsMap(pools)
	if err != nil {
		return "", fmt.Errorf("could not EncodePoolSimulatorsMap: %w", err)
	}
	log.Ctx(ctx).Info().Msgf("publishing %d pools, encoded size = %s, encoding took %s",
		len(pools), humanize.Bytes(uint64(len(encoded))), time.Since(start).String())

	start = time.Now()
	result, err := p.aevmClient.StorePreparedPools(ctx, &types.StorePreparedPoolsParams{
		EncodedPools: encoded,
	})
	if err != nil {
		return "", fmt.Errorf("could not StorePreparedPools: %w", err)
	}
	log.Ctx(ctx).Info().Msgf("done publishing %d pools, took = %s", len(pools), time.Since(start).String())

	addrs := make(map[string]struct{})
	for addr := range pools {
		addrs[addr] = struct{}{}
	}
	p.addAndCleanupPublishedPools(result.StorageID, addrs, pools)

	return result.StorageID, nil
}

// Add new pools map and remove olds pools maps so that it contains at most `numStorageIDsToRetain` pools maps.
func (p *PoolsPublisher) addAndCleanupPublishedPools(storageID string, addrs map[string]struct{}, pools map[string]poolpkg.IPoolSimulator) {
	p.publishedPoolIDs.Store(storageID, addrs)
	p.publishedPools.Store(storageID, pools)

	var toRemove []string
	p.storageIDsMu.Lock()
	p.storageIDs = append(p.storageIDs, storageID)
	if len(p.storageIDs) > p.numStorageIDsToRetain {
		toRemove = append([]string(nil), p.storageIDs[:len(p.storageIDs)-p.numStorageIDsToRetain]...)
		p.storageIDs = append([]string(nil), p.storageIDs[len(p.storageIDs)-p.numStorageIDsToRetain:]...)
	}
	p.storageIDsMu.Unlock()

	for _, storageID := range toRemove {
		p.publishedPoolIDs.Delete(storageID)
	}
	for _, storageID := range toRemove {
		p.publishedPools.Delete(storageID)
	}
}
