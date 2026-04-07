package kipseliprop

import (
	"errors"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	DexType    = "kipseli-prop"
	defaultGas = 125_000
	sampleSize = 15 // power-of-10 levels
)

var maxInSampleBps = []int{
	1000, 1500, 2200, 3200, 4000, // 10–40%
	4500, 5000, 5600, 6200, 6800, // 40–68%
	7300, 7900, 8500, 9100, 9900, // 73–99%
}

var (
	DomainType = apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"PropAmmVerification": []apitypes.Type{
				{Name: "tokenIn", Type: "address"},
				{Name: "tokenOut", Type: "address"},
				{Name: "timestampInMilisec", Type: "uint256"},
			},
		},
		PrimaryType: "PropAmmVerification",
		Domain: apitypes.TypedDataDomain{
			Name:    "VerificationImpl",
			Version: "1",
		},
	}

	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
