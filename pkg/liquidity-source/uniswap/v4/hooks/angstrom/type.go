package angstrom

import "github.com/ethereum/go-ethereum/common"

type RFQExtra struct {
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
