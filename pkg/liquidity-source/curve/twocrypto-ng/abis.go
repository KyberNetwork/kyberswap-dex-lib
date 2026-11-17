package twocryptong

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	CurveTwocryptoNGABI abi.ABI
	// YBTwocryptoPolicyABI is the Yield Basis twocrypto policy's views (see TwocryptoPolicyState).
	YBTwocryptoPolicyABI abi.ABI
	// YBPolicyABI is YBOraclizedPolicy's views (see PolicyState).
	YBPolicyABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&CurveTwocryptoNGABI, curveTwocryptoNGABIBytes},
		{&YBTwocryptoPolicyABI, ybTwocryptoPolicyABIBytes},
		{&YBPolicyABI, ybPolicyABIBytes},
	}

	var err error
	for _, b := range build {
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
