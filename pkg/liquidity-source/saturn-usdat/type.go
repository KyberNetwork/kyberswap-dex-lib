package saturnusdat

import "github.com/holiman/uint256"

// StaticExtra is immutable pool data.
type StaticExtra struct {
	Facility string `json:"facility"`
	Asset    string `json:"asset"`
}

// Extra is the mutable on-chain state read by the tracker. Pause flags and the
// approved-extension flag gate the facility. The whitelist applies to the caller and the
// recipient, both the configured executor on a single hop; NotWhitelisted is set only when
// an executor is configured. Freeze checks are per account and not modelled.
type Extra struct {
	AssetCap          *uint256.Int `json:"cap"`
	AssetBalance      *uint256.Int `json:"bal"`
	AssetDecimals     uint8        `json:"aDec"`
	ExtensionDecimals uint8        `json:"eDec"`
	AssetAllowed      bool         `json:"allowed"`
	FacilityPaused    bool         `json:"fPaused,omitempty"`
	USDatPaused       bool         `json:"uPaused,omitempty"`
	ExtensionApproved bool         `json:"approved"`
	WhitelistEnabled  bool         `json:"wl,omitempty"`
	NotWhitelisted    bool         `json:"notWl,omitempty"`
}

type SwapInfo struct {
	// AssetAmount is the USDC actually pulled (input minus decimals dust).
	AssetAmount *uint256.Int `json:"-"`
}

type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}
