package lunarbase

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const snapshotCacheCapacity = 64

type snapshotReference struct {
	number uint64
	hash   common.Hash
}

type snapshotCacheKey struct {
	chainID valueobject.ChainID
	address common.Address
	snapshotReference
}

// Only complete, verified states enter this bounded, tracker-local cache.
// A block hash identifies immutable state, but is not proof that the block
// is still canonical. Every reuse must follow a new canonical header read.
type snapshotCache struct {
	mu      sync.Mutex
	entries map[snapshotCacheKey]*rpcState
	order   []snapshotCacheKey
}

func cloneRPCState(state *rpcState) *rpcState {
	copyState := *state
	copyState.reserveX = new(big.Int).Set(state.reserveX)
	copyState.reserveY = new(big.Int).Set(state.reserveY)
	copyState.extra.SqrtPriceX96 = new(uint256.Int).Set(state.extra.SqrtPriceX96)
	return &copyState
}

// Capture copies before starting the RPC that will validate their canonical
// membership. Entries inserted by concurrent refreshes after this point must
// not borrow a header check which started before those entries were read.
func (c *snapshotCache) capture(chainID valueobject.ChainID, address common.Address) map[snapshotReference]*rpcState {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var states map[snapshotReference]*rpcState
	for key, state := range c.entries {
		if key.chainID == chainID && key.address == address {
			if states == nil {
				states = make(map[snapshotReference]*rpcState)
			}
			states[key.snapshotReference] = cloneRPCState(state)
		}
	}
	return states
}

func (c *snapshotCache) put(chainID valueobject.ChainID, address common.Address, state *rpcState) {
	if c == nil {
		return
	}
	key := snapshotCacheKey{chainID, address, snapshotReference{state.blockNumber, state.blockHash}}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[snapshotCacheKey]*rpcState)
	}
	if _, exists := c.entries[key]; !exists {
		if len(c.order) == snapshotCacheCapacity {
			delete(c.entries, c.order[0])
			c.order = c.order[1:]
		}
		c.order = append(c.order, key)
	}
	c.entries[key] = cloneRPCState(state)
}
