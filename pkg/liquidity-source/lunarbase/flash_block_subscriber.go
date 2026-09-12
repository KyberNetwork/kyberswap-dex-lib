package lunarbase

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Kept for source compatibility with callers inspecting GetLatestState.
type poolState struct {
	SqrtPriceX96      *uint256.Int
	FeeAskX24         uint32
	FeeBidX24         uint32
	ReserveX          *uint256.Int
	ReserveY          *uint256.Int
	LatestUpdateBlock uint64
	BlockDelay        uint64
	ConcentrationK    uint32
	MaxPunishmentX24  uint32
	BlockNumber       uint64
	// Legacy cache fields remain readable to source-level callers, although
	// this retired cache never returns a state for quotation.
	HasBlockDelay         bool
	HasConcentrationK     bool
	HasMaxPunishmentX24   bool
	BlockDelayBlock       uint64
	ConcentrationKBlock   uint64
	MaxPunishmentX24Block uint64

	StateUpdatedAt    time.Time
	ReservesUpdatedAt time.Time
}

// IsStale always rejects the retired partial event cache.
func (s *poolState) IsStale() bool { return true }

// FlashBlockSubscriber is retained for source compatibility.
// Deprecated: independent event streams cannot establish a complete snapshot.
// Use PoolTracker.GetNewPoolState, which verifies one canonical RPC snapshot.
type FlashBlockSubscriber struct{}

// InitFlashBlockSubscriber no longer starts background connections.
// Deprecated: quote state is refreshed by PoolTracker through RPC.
func InitFlashBlockSubscriber(wsURL, flashWsURL string, pools []common.Address) {}

// GetFlashBlockSubscriber returns nil: partial event state is not quoteable.
// Deprecated: use PoolTracker.GetNewPoolState.
func GetFlashBlockSubscriber() *FlashBlockSubscriber { return nil }

// GetLatestState returns nil, including when called on a nil receiver.
// Deprecated: use PoolTracker.GetNewPoolState.
func (s *FlashBlockSubscriber) GetLatestState() *poolState { return nil }
