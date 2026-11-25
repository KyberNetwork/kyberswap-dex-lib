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
	common.HexToAddress("0x7e00422559c4c6a9b1592a25074A420a96412088"), // old base
	common.HexToAddress("0xecC495C00288018e4F4c61315163F598f96D2088"), // base
	common.HexToAddress("0x6fbbda42a6c7e4b1e413af98cf249e4276586088"), // arbitrum
}

// HookV1Addresses are the V1 hook (ladder packed in two words, cursor, inventory, oracle bound).
var HookV1Addresses = []common.Address{
	common.HexToAddress("0x09b846cadee474ee290db4bb169f156200d92088"), // base
	common.HexToAddress("0x0da3f5af21715a2e74adf11ac0b04d9aca842088"), // arbitrum
}

var (
	ErrYield       = errors.New("aegisprop: hook yields, swap goes to the core pool")
	ErrPartialFill = errors.New("aegisprop: hook fills only part of the exact-in amount")
	ErrUnsupported = errors.New("aegisprop: pool oracle config is not modelled")
	ErrNoInversion = errors.New("aegisprop: cannot invert exact-in for the requested output")
)

// Integer average of 8 measured beforeSwap calls (4 per chain, both directions, 113061..117898).
const gasBeforeSwapV1 int64 = 115444
