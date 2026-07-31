package aegisprop

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	gasBeforeSwap int64  = 75160
	bpsDenom      uint64 = 10_000
)

var (
	ErrInvalidAmount           = errors.New("aegisprop: invalid swap amount")
	ErrNoLiquidity             = errors.New("aegisprop: ladder has no active levels for this side")
	ErrStaleLadder             = errors.New("aegisprop: ladder quote has expired")
	ErrInsufficientLadderDepth = errors.New("aegisprop: ladder depth insufficient for requested amount")
)

var HookAddresses = []common.Address{
	common.HexToAddress("0x7e00422559c4c6a9b1592a25074a420a96412088"), // base
}
