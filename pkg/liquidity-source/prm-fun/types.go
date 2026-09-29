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

// StaticExtra is immutable per-pool data, set at discovery. Pools exist only for
// ETH-paired memes; stock-paired ones are out of scope for this pool type.
type StaticExtra struct {
	RouterAddress  string `json:"rA"`
	CurveAddress   string `json:"cA"`
	MemeToken      string `json:"mT"`
	GraduationDesk string `json:"gD"`
}

type Extra struct {
	Phase       uint8        `json:"ph"`
	VirtualMeme *uint256.Int `json:"vM"`
	VirtualDesk *uint256.Int `json:"vD"`
	MemeSold    *uint256.Int `json:"mS"`
	DeskRaised  *uint256.Int `json:"dR"`
}

// SwapInfo carries swap direction to the encoder, which cannot infer it from the token
// addresses: the desk side is listed as wrapped native.
type SwapInfo struct {
	IsBuy        bool   `json:"iB"`
	CurveAddress string `json:"cA"`
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

// PoolMeta is returned by GetMetaInfo. ApprovalAddress is the PremiumRouter proxy, which a
// sell must approve meme-token spending to; buys are payable and need no approval.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	BlockNumber     uint64 `json:"blockNumber"`
}

type GetReservesResult struct {
	QuoteReserve *big.Int
	TokenReserve *big.Int
}
