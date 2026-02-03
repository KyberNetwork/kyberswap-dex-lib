package stabull

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type StaticExtra struct {
	Oracles [2]common.Address `json:"o"` // [base, quote]
}

// Extra contains additional pool state that gets serialized
type Extra struct {
	// Curve parameters from viewCurve()
	CurveParams `json:"c"`
	// Oracle rates for both tokens (base/USD and USDC/USD)
	OracleRates [2]*uint256.Int `json:"os"` // e.g., [NZD/USD,USDC/USD] from Chainlink
	// Derived oracle rate (baseOracleRate / quoteOracleRate)
	OracleRate *uint256.Int `json:"o,omitempty"`
}

// CurveParams represents the Stabull curve parameters defining the shape of the pricing curve
type CurveParams struct {
	Alpha   *uint256.Int `json:"a"`
	Beta    *uint256.Int `json:"b"`
	Delta   *uint256.Int `json:"d"`
	Epsilon *uint256.Int `json:"e"`
	Lambda  *uint256.Int `json:"l"`
}
