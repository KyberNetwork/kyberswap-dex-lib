package unipool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Extra is the dynamic per-pool state serialized into entity.Pool.Extra.
//
// All values are raw storage snapshots from the pair (not projected to current time):
// the simulator interpolates the virtual reserves forward at quote time using
// LastUpdateTimestamp + PriceDecay, mirroring the on-chain previewVirtualReservesElapsed.
type Extra struct {
	Reserve0 *big.Int `json:"reserve0"`
	Reserve1 *big.Int `json:"reserve1"`

	VirtualReserve0In  *big.Int `json:"vr0In"`
	VirtualReserve0Out *big.Int `json:"vr0Out"`
	VirtualReserve1In  *big.Int `json:"vr1In"`
	VirtualReserve1Out *big.Int `json:"vr1Out"`

	LastUpdateTimestamp uint64 `json:"lastUpdateTs"`
	PriceDecay          uint64 `json:"priceDecay"`

	FeeLpBps   uint16 `json:"feeLpBps"`
	FeePoolBps uint16 `json:"feePoolBps"`

	TotalBorrowed0 *big.Int `json:"totalBorrowed0"`
	TotalBorrowed1 *big.Int `json:"totalBorrowed1"`

	// SwapPriceToleranceBps caps the spread between the AMM mid-price and the
	// buy/sell prices implied by the swap. Disabled if equal to math.MaxUint16.
	SwapPriceToleranceBps uint16 `json:"swapPriceToleranceBps"`
}

// StaticExtra is the immutable per-pool data serialized into entity.Pool.StaticExtra.
type StaticExtra struct {
	FactoryAddress string `json:"factoryAddress"`
}

// zeroExtra returns an Extra with all big.Int fields freshly allocated to 0.
// Used at pool discovery time (factory event or list updater) when the real
// state hasn't been tracked yet.
func zeroExtra() Extra {
	return Extra{
		Reserve0:           big.NewInt(0),
		Reserve1:           big.NewInt(0),
		VirtualReserve0In:  big.NewInt(0),
		VirtualReserve0Out: big.NewInt(0),
		VirtualReserve1In:  big.NewInt(0),
		VirtualReserve1Out: big.NewInt(0),
		TotalBorrowed0:     big.NewInt(0),
		TotalBorrowed1:     big.NewInt(0),
	}
}

// --- ABI receiver structs ---------------------------------------------------

// virtualReservesABI mirrors the Solidity `VirtualReserves` struct returned by
// pair.getVirtualReserves(). Field names match the tuple component names so the
// go-ethereum ABI unpacker can map them by PascalCase.
type virtualReservesABI struct {
	VirtualReserve0In  *big.Int
	VirtualReserve0Out *big.Int
	VirtualReserve1In  *big.Int
	VirtualReserve1Out *big.Int
}

// reservesABI receives pair.getReserves() => (uint256 reserve0, uint256 reserve1).
type reservesABI struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

// feesBpsABI receives pair.getFeesBps() => (uint16 feeLpBps, uint16 feePoolBps, uint16 burnFeeBps).
type feesBpsABI struct {
	FeeLpBps   uint16
	FeePoolBps uint16
	BurnFeeBps uint16
}

// tokensABI receives pair.getTokens() => (address token0, address token1).
type tokensABI struct {
	Token0 common.Address
	Token1 common.Address
}
