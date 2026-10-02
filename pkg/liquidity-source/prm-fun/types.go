package prmfun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// GetMemeResult matches MemeFactory.getMeme(address).
type GetMemeResult struct {
	Data struct {
		SubjectId      [32]byte
		Token          common.Address
		Curve          common.Address
		Creator        common.Address
		GraduationDesk *big.Int
		CreatedAt      uint64
	}
}

// StaticExtra is immutable per-pool data for ETH, PRM and stock-paired memes.
type StaticExtra struct {
	RouterAddress  string `json:"rA"`
	CurveAddress   string `json:"cA"`
	MemeToken      string `json:"mT"`
	GraduationDesk string `json:"gD"`
	IsNativeQuote  bool   `json:"nQ"`
}

type Extra struct {
	Phase       uint8        `json:"ph"`
	VirtualMeme *uint256.Int `json:"vM"`
	VirtualDesk *uint256.Int `json:"vD"`
	MemeSold    *uint256.Int `json:"mS"`
	DeskRaised  *uint256.Int `json:"dR"`
}

// SwapInfo identifies settlement and stores the exact post-swap state. The input
// offered to CalcAmountOut can exceed the amount actually accepted by the curve.
type SwapInfo struct {
	IsBuy         bool   `json:"iB"`
	CurveAddress  string `json:"cA"`
	IsNativeQuote bool   `json:"nQ"`
	NewState      Extra  `json:"-"`
}

type PoolsListUpdaterMetadata struct {
	Offset  int `json:"offset"`
	Version int `json:"version"`
}

// PoolMeta is returned by GetMetaInfo. ApprovalAddress is the PremiumRouter proxy, which a
// sell must approve meme-token spending to. ERC-20 pair buys also require approval;
// only ETH-pair buys send native value.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	BlockNumber     uint64 `json:"blockNumber"`
	IsNativeQuote   bool   `json:"isNativeQuote"`
}

type GetReservesResult struct {
	QuoteReserve *big.Int
	TokenReserve *big.Int
}
