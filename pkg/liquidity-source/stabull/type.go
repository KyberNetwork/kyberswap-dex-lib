package stabull

import (
	"math/big"
)

// Metadata tracks the last processed block for event scanning
type Metadata struct {
	LastBlock uint64 `json:"lastBlock"`
}

// CurveParameters represents the Stabull curve parameters
// These define the shape of the pricing curve
// Fields are strings for JSON serialization (standard for big numbers)
type CurveParameters struct {
	Alpha   string `json:"alpha"`   // Curve parameter
	Beta    string `json:"beta"`    // Curve parameter
	Delta   string `json:"delta"`   // Curve parameter
	Epsilon string `json:"epsilon"` // Curve parameter
	Lambda  string `json:"lambda"`  // Curve parameter
}

// Reserves represents the pool's token reserves
type Reserves struct {
	Reserve0 *big.Int // Base token (non-USDC stablecoin)
	Reserve1 *big.Int // Quote token (USDC)
}

// Extra contains additional pool state that gets serialized
// All fields are strings for JSON compatibility
type Extra struct {
	// Curve parameters from viewCurve()
	CurveParams CurveParameters `json:"curveParams"`

	// Oracle addresses for Chainlink price feeds
	BaseOracleAddress  string `json:"baseOracleAddress,omitempty"`  // Chainlink aggregator for base token (e.g., NZD/USD)
	QuoteOracleAddress string `json:"quoteOracleAddress,omitempty"` // Chainlink aggregator for quote token (USDC/USD)

	// Oracle rates for both tokens (base/USD and USDC/USD)
	// Strings for JSON serialization
	BaseOracleRate  string `json:"baseOracleRate,omitempty"`  // e.g., NZD/USD from Chainlink
	QuoteOracleRate string `json:"quoteOracleRate,omitempty"` // USDC/USD from Chainlink

	// Derived oracle rate (baseOracleRate / quoteOracleRate)
	OracleRate string `json:"oracleRate,omitempty"`
}

// Gas represents gas costs for different operations
type Gas struct {
	Swap int64
}

// Meta provides metadata about the pool for the aggregator
type Meta struct {
	Alpha      string `json:"alpha"`
	Beta       string `json:"beta"`
	Delta      string `json:"delta"`
	Epsilon    string `json:"epsilon"`
	Lambda     string `json:"lambda"`
	OracleRate string `json:"oracleRate,omitempty"`
}

// ViewCurveResult represents the return value from viewCurve()
type ViewCurveResult struct {
	Alpha   *big.Int
	Beta    *big.Int
	Delta   *big.Int
	Epsilon *big.Int
	Lambda  *big.Int
}
