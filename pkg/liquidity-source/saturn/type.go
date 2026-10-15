package saturn

import "github.com/holiman/uint256"

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	SwapFacilityPaused  bool         `json:"swapFacilityPaused"`
	USDatPaused         bool         `json:"usdatPaused"`
	ExtensionApproved   bool         `json:"extensionApproved"`
	AssetAllowed        bool         `json:"assetAllowed"`
	AssetCap            *uint256.Int `json:"assetCap"`
	AssetBalance        *uint256.Int `json:"assetBalance"`
	AssetDecimals       uint8        `json:"assetDecimals"`
	WhitelistEnabled    bool         `json:"whitelistEnabled"`
	ExecutorWhitelisted bool         `json:"executorWhitelisted"`
}

type StaticExtra struct {
	Executor string `json:"executor"`
	Asset    string `json:"asset"`
	USDat    string `json:"usdat"`
	Gas      Gas    `json:"gas,omitempty"`
}

type SwapInfo struct {
	EffectiveAssetAmount *uint256.Int `json:"effectiveAssetAmount"`
	ExtensionAmount      *uint256.Int `json:"extensionAmount"`
}

type Gas struct {
	Mint int64 `json:"mint"`
}
