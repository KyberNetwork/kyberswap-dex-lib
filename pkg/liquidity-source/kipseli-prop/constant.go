package kipseliprop

import (
	"errors"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	DexType    = "kipseli-prop"
	defaultGas = 125_000
	sampleSize = 15
)

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
