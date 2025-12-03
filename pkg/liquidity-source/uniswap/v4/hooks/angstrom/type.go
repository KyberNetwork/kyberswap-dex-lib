package angstrom

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type HookExtra struct {
	UnlockedFee         *big.Int `json:"uFee"`
	ProtocolUnlockedFee *big.Int `json:"pFee"`

	LatestAttestations     []Attestation `json:"latestAttestations,omitempty"`
	LatestAttestationsTime int64         `json:"latestAttestationsTime,omitempty"`
}

type SwapInfo struct {
	Adapter      common.Address
	Attestations []Attestation
}

type Attestation struct {
	BlockNumber int    `json:"blockNumber"`
	UnlockData  string `json:"unlockData"`
}

type AttestationsResponse struct {
	Success      bool          `json:"success"`
	Attestations []Attestation `json:"attestations"`
}
