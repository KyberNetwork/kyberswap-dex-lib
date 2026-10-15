package ilyrislb

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// chainReader is the tracker's view of the chain; rpc_ethrpc.go binds it to ethrpc and the
// tests to a fake book.
type chainReader interface {
	// PoolState reads lens metadata, getTotalFeeRate() and the fee config/state in one
	// multicall, plus that block's timestamp.
	PoolState(ctx context.Context, pool string) (RawPoolState, error)
	// ScanBitmap reads getBitmapWord per word and nextNonEmptyBin per probe in one multicall.
	ScanBitmap(ctx context.Context, pool string, blockNumber uint64, words []int32,
		probes []binProbe) ([]*big.Int, []binProbeResult, error)
	// BinReserves reads getBinReserves(ids) in one multicall.
	BinReserves(ctx context.Context, pool string, blockNumber uint64, ids []int32) ([]RawBin, error)
	// GuardState reads the guard flags and dry-runs checkSwap both ways at blockNumber.
	GuardState(ctx context.Context, guard, pool string, activeID int32, blockNumber uint64) (RawGuardState, error)
	// FactoryPools reads BinFactory.allPools[offset:offset+limit] and each pool's tokens.
	FactoryPools(ctx context.Context, factory string, offset, limit int) ([]FactoryPool, int, error)
}

type RawPoolState struct {
	BinStepBps           uint32
	DecimalsX, DecimalsY uint8
	ActiveID             int32
	TotalFeeRate         uint64
	Fee                  FeeParams
	MarketGuard          common.Address
	Bins                 []RawBin
	BlockNumber          uint64
	BlockTimestamp       uint64
}

type RawBin struct {
	ID                 int32
	ReserveX, ReserveY *big.Int
}

type RawGuardState struct {
	SwapsPaused bool // swapsPaused(), or checkSwap reverted
	FreezeStart uint64
	FreezeEnd   uint64
}

type FactoryPool struct {
	Address, TokenX, TokenY string
}

type binProbe struct {
	From  int32
	XForY bool
}

type binProbeResult struct {
	Found bool
	ID    int32
}
