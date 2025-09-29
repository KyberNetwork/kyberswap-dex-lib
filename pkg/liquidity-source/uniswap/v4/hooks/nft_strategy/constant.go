package nftstrat

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var HookAddresses = []common.Address{
	common.HexToAddress("0xe3C63A9813Ac03BE0e8618B627cb8170cfA468c4"),
	common.HexToAddress("0x5d8A61FA2Ced43EEaBffC00c85f705E3e08c28c4"),
}

var (
	ErrHookExtraNotFound = errors.New("hook extra not found")
)
