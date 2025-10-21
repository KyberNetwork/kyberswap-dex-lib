package angstrom

import "github.com/ethereum/go-ethereum/common"

type RFQExtra struct {
	Adapter      common.Address
	Attestations []Attenstation
}

type Attenstation struct {
	BlockNumber int
	UnlockData  string
}

type AttenstationsResponse struct {
	Success      bool           `json:"success"`
	Attestations []Attenstation `json:"attestations"`
}
