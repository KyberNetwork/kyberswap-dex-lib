package st0x

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	gasBeforeSwap int64  = 39838
	bpsDenom      uint64 = 10_000
)

var (
	ErrNoPriceSet      = errors.New("st0x: oracle has no price for pool")
	ErrStalePrice      = errors.New("st0x: oracle price is stale")
	ErrInvalidSpread   = errors.New("st0x: spread exceeds denominator")
	ErrInsufficientRsv = errors.New("st0x: hook reserve insufficient for output")
)

var (
	hookAddress   = common.HexToAddress("0x9AF1a97021B92c47219D2fb24DeBA51C248A2a88")
	oracleAddress = common.HexToAddress("0x42fA16d1e9f1A1152a3d8408829684FBCF885E69")

	HookAddresses = []common.Address{hookAddress}
)
