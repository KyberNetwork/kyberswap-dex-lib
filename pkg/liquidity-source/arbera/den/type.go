package arberaden

import "github.com/holiman/uint256"

type Extra struct {
	Assets        []Asset        `json:"assets"`
	AssetSupplies []*uint256.Int `json:"assetSupplies"`
	Supply        *uint256.Int   `json:"supply"`
	Fee           Fee            `json:"fee"`
}

type Asset struct {
	Token           string       `json:"token"`
	Weighting       *uint256.Int `json:"weighting"`
	BasePriceUSDX96 *uint256.Int `json:"basePriceUSDX96"`
	C1              string       `json:"c1"`
	Q1              *uint256.Int `json:"q1"`
}

// all fees: 1 == 0.01%, 10 == 0.1%, 100 == 1%
type Fee struct {
	Bond   *uint256.Int `json:"bond"`
	Debond *uint256.Int `json:"debond"`
	Burn   *uint256.Int `json:"burn"`
}
