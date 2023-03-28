package l2feecalculator

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	// In order to estimate the L1 fee for L2 blockchains, we need to have the length of the calldata
	// Thus, we need to create a raw transaction, and to create a raw transaction, we will need
	// private key, to address, tx value and tx gas limit

	// DummyPrivateKey is just a random private key with no money inside
	DummyPrivateKey = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	DummyToAddress  = "0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d"
	DummyValue      = 1000000000000000000
	DummyGasLimit   = 21000
)

var (
	privateKey *ecdsa.PrivateKey
)

func init() {
	pk, err := crypto.HexToECDSA(DummyPrivateKey)
	if err != nil {
		// should stop the instance immediately
		logger.Fatalf("failed to get private key, err: %v", err)
	}

	privateKey = pk
}
