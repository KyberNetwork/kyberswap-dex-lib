package fermi

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type StateOverrides map[string]map[string]string

type Extra struct {
	Fermi            string          `json:"fermi"`
	TraderVault      string          `json:"traderVault"`
	BlockNumber      uint64          `json:"bn"`
	LastUpdatedBlock uint64          `json:"lub,omitempty"`
	Curve            *CurveData      `json:"curve,omitempty"`
	StateOverrides   *StateOverrides `json:"so,omitempty"`
}

type CurveData struct {
	MidPrice           string `json:"mp"`
	FeeBaseBps         uint16 `json:"fb"`
	SafetyFeeBps       uint16 `json:"sf"`
	ScalingDenominator string `json:"sd"`
	MaxAmountIn        string `json:"max"`
	TokenInDecScale    string `json:"ds0"`
	TokenOutDecScale   string `json:"ds1"`
	SizeSpline         []Knot `json:"sp"`
	InventorySpline    []Knot `json:"ip"`
	VaultReserve0      string `json:"vr0"`
	VaultReserve1      string `json:"vr1"`
}

// Knot is one control point of a piecewise-cubic spline. On-chain layout
// (three packed 256-bit slots per knot):
//
//	slot+0: XLo : int128 | XHi : int128   bracket bounds in 1e18 space
//	slot+1: C0  : int128 | C1  : int128   constant + linear coef
//	slot+2: C2  : int128 | C3  : int128   quadratic + cubic coef
//
// Polynomial: y = C0*1e18 + C1*t + C2*(t²/1e18) + C3*(t³/1e18), t ∈ [0, 1e18].
// Matches FermiEngine private function 0x3dcb.
type Knot struct {
	XLo string `json:"xl"`
	XHi string `json:"xh"`
	C0  string `json:"c0"`
	C1  string `json:"c1"`
	C2  string `json:"c2"`
	C3  string `json:"c3"`
}

type StaticExtra struct {
	FermiSwapper string `json:"fS"`
}

type PoolMeta struct {
	BlockNumber    uint64          `json:"bn"`
	FermiSwapper   string          `json:"fS"`
	StateOverrides *StateOverrides `json:"so,omitempty"`
}

type Config struct {
	DexId        string              `json:"dexId"`
	ChainId      valueobject.ChainID `json:"chainId"`
	FermiSwapper string              `json:"fermiSwapper"`

	Titan TitanConfig `json:"titan"`
}
