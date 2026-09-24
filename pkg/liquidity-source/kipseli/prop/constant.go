package prop

import (
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

const (
	DexType    = "kipseli-prop"
	defaultGas = 125_000

	// maxAge bounds how long a probed ladder may be quoted against before a
	// fresher one is required — mirrors titan-prop's freshnessTTL.
	maxAge = 30 * time.Second
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

	ErrInvalidToken          = kipseli.ErrInvalidToken
	ErrInsufficientLiquidity = kipseli.ErrInsufficientLiquidity
)
