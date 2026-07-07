package stabull

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "stabull"

	// Pool (Curve) methods
	poolMethodNumeraires  = "numeraires"  // numeraires(uint256) returns token address at index
	poolMethodAssimilator = "assimilator" // assimilator(address) returns assimilator address for token
	poolMethodCurve       = "curve"       // curve() returns (alpha, beta, delta, epsilon, lambda, totalSupply)

	// Assimilator methods
	assimilatorMethodGetRate = "getRate" // getRate() returns oracle rate

	// Default gas costs for Stabull swaps
	// Stabull uses curve math + oracle checks, higher than simple AMMs
	defaultGas = 204523 // Approximate gas for originSwap/targetSwap with oracle checks
)

var (
	Weight50             = new(uint256.Int).Div(big256.U2Pow64, big256.U2)
	ConvergencePrecision = big256.TenPow(13)
	OracleDecimals       = big256.TenPow(8)
)
