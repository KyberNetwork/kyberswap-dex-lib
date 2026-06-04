package altfun

import "github.com/holiman/uint256"

// Lifecycle mirrors Bonding.Lifecycle enum: 0=Curve, 1=Graduating, 2=Graduated.
type Lifecycle uint8

const (
	LifecycleCurve      Lifecycle = 0
	LifecycleGraduating Lifecycle = 1
	LifecycleGraduated  Lifecycle = 2
)

// Extra holds per-block dynamic state: only the bonding-curve Pair state.
// All BounceTech LT pricing (exchangeRate, fees, mintPaused, etc.) is delegated
// to the bounce-tech base pool via the IMetaPoolSimulator pattern.
type Extra struct {
	ReserveToken *uint256.Int `json:"reserveToken"`
	ReserveAsset *uint256.Int `json:"reserveAsset"` // virtual LT reserve in Pair
	K            *uint256.Int `json:"k"`
	TokenBalance *uint256.Int `json:"tokenBalance"` // actual ERC20 balance in Pair (buy cap)

	Lifecycle Lifecycle `json:"lifecycle"`
}

// StaticExtra holds immutable per-pool data set once at discovery time.
type StaticExtra struct {
	PairAddress string `json:"pairAddress"`
	// LTAddress is the BounceTech LT used as the bonding-curve reserve asset.
	// It also serves as the bounce-tech pool address in the basePoolMap.
	LTAddress  string `json:"ltAddress"`
	USDC       string `json:"usdc"`
	ZapAddress string `json:"zapAddress"`
	BuyFeeBps  uint64 `json:"buyFeeBps"`
	SellFeeBps uint64 `json:"sellFeeBps"`
	// BasePool is the bounce-tech pool address for this meme token's LT.
	BasePool string `json:"basePool"`
	// GraduationThresholdUsd is Bonding.graduationThresholdUsd() (18-dec, immutable).
	// Used to compute ltUntilGraduation dynamically from the base pool's exchangeRate.
	GraduationThresholdUsd *uint256.Int `json:"graduationThresholdUsd"`
}

// SwapInfo carries per-swap data for executor calldata and UpdateBalance.
type SwapInfo struct {
	Pool     string `json:"pool"`
	IsSell   bool   `json:"isSell"`
	Referrer string `json:"referrer"`
	// Local-only fields (not encoded into calldata).
	AmountInUsed  *uint256.Int `json:"-" msgpack:"-"`
	BaseToConvert *uint256.Int `json:"-" msgpack:"-"`
}
