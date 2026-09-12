package ilyris

import (
	"context"
	"math/big"
)

// chainReader is everything this adapter needs from the chain, and nothing else.
//
// The tracker depends on THIS rather than on *ethrpc.Client so the parts that actually carry
// bugs -- cold-start detection, folding logs into the book, propagating guard state -- can be
// tested against a fake instead of a live node. Their own trackers take the concrete client,
// which is why their tracker tests are the thinnest part of the template.
//
// The ethrpc implementation is a thin binding over this in rpc_ethrpc.go; it holds no logic,
// so there is nothing in it to test that a live call would not have to prove anyway.
type chainReader interface {
	// PoolState is one BinPoolLens.getPoolState(pool, radius) call.
	//
	// NOTE the radius is uint24 on chain, NOT uint16. Encoding it as uint16 reverts with
	// empty return data, which reads as a broken lens rather than a wrong ABI -- it cost a
	// probe during the research pass and it will cost an integrator one too.
	PoolState(ctx context.Context, pool string, radius uint32) (RawPoolState, error)

	// GuardState reads the market guard at blockNumber so the gate and the bin
	// book are the same block. Separate call because the guard is a separate
	// contract and is owner-mutable, so its ADDRESS must be re-read every refresh
	// rather than cached from pool creation.
	GuardState(ctx context.Context, guard string, blockNumber uint64) (RawGuardState, error)

	// FactoryPools enumerates from BinFactory: allPoolsLength() then allPools(i).
	FactoryPools(ctx context.Context, factory string, offset, limit int) (pools []string, total int, err error)
}

// RawPoolState is one lens read, pinned to a single block.
type RawPoolState struct {
	TokenX, TokenY       string
	DecimalsX, DecimalsY uint8
	BinStepBps           uint32
	ActiveID             int32
	TotalFeeRate         uint64
	MarketGuard          string
	Bins                 []RawBin
	BlockNumber          uint64
	BlockTimestamp       uint64
}

type RawBin struct {
	ID       int32
	ReserveX *big.Int
	ReserveY *big.Int
}

// RawGuardState is what decides whether a quote may be offered at all.
type RawGuardState struct {
	SwapsPaused bool
	FreezeEnd   uint64
	BlockNumber uint64
}
