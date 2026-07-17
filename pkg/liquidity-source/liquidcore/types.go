package liquidcore

import (
	"github.com/ethereum/go-ethereum/common"
)

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}
