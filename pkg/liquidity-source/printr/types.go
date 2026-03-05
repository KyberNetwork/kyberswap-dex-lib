package printr

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// GetCurveResult matches the return type of getCurve(address) on the Printr contract.
// The getter returns CurveInfo with already-expanded fields (not the packed Curve struct).
type GetCurveResult struct {
	BasePair            common.Address
	TotalCurves         uint16
	MaxTokenSupply      *big.Int
	VirtualReserve      *big.Int
	Reserve             *big.Int
	CompletionThreshold *big.Int
}

// StaticExtra stores immutable per-pool data (set once at pool creation).
type StaticExtra struct {
	PrintrAddr     string `json:"pA"`
	Token          string `json:"tk"`
	BasePair       string `json:"bP"`
	TotalCurves    uint16 `json:"tC"`
	MaxTokenSupply string `json:"mTS"`
	VirtualReserve string `json:"vR"`
}

// Extra stores mutable per-pool state (refreshed by the tracker).
type Extra struct {
	Reserve             *uint256.Int `json:"r"`
	CompletionThreshold *uint256.Int `json:"cT"`
	TradingFee          uint16       `json:"tF"`
	Paused              bool         `json:"p"`
}

// SwapInfo is attached to CalcAmountOutResult for the executor and UpdateBalance.
type SwapInfo struct {
	IsBuy bool `json:"iB"`

	// Pre-computed state for UpdateBalance (not serialized to executor)
	reserveDelta *uint256.Int // exact amount to add/subtract from reserve
}

// TokenListResponse matches the Uniswap-standard tokenlist JSON format
// served by PRINTR's API at /chains/{chainId}/tokenlist.json.
type TokenListResponse struct {
	Name    string           `json:"name"`
	Tokens  []TokenListEntry `json:"tokens"`
	Version TokenListVersion `json:"version"`
}

type TokenListEntry struct {
	ChainId    int                    `json:"chainId"`
	Address    string                 `json:"address"`
	Name       string                 `json:"name"`
	Symbol     string                 `json:"symbol"`
	Decimals   int                    `json:"decimals"`
	Extensions map[string]interface{} `json:"extensions"`
}

type TokenListVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// MetaInfo is returned by GetMetaInfo for the router.
type MetaInfo struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

// PoolsListUpdaterMetadata tracks incremental pool discovery state.
type PoolsListUpdaterMetadata struct {
	VersionMajor int `json:"vMaj"`
	VersionMinor int `json:"vMin"`
	VersionPatch int `json:"vPat"`
}
