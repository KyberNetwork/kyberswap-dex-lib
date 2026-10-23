package hyfiprop

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// gasDirect is the executor's executeGeneric frame (minus the approve) for HyFi.swapExactInDirect on
// Base: the integer average of 10 Tenderly route simulations, 5 per direction (USDC in: 5x60568,
// NVDAc in: 5x79325), via test-swap's poolGasUsed. The v4 simulator adds DefaultGas.BaseGas for the
// PoolManager swap, which the direct path never runs.
const gasDirect int64 = (5*60568 + 5*79325) / 10

const (
	numTicks  = 68 // ticks per book side
	headTicks = 4  // ticks stored in slot0
	wordTicks = 32 // ticks per full word
	pips      = 1_000_000
)

var (
	ErrInvalidAmount         = errors.New("hyfi: invalid swap amount")
	ErrPairNotConfigured     = errors.New("hyfi: pair not configured")
	ErrInsufficientLiquidity = errors.New("hyfi: book exhausted")
	ErrInsufficientInv       = errors.New("hyfi: contract balance insufficient")
	ErrOverflow              = errors.New("hyfi: arithmetic overflow")
	ErrZeroOutput            = errors.New("hyfi: zero output")
	ErrUnexpectedResponse    = errors.New("hyfi: unexpected RPC response")
)

// HookAddresses are the HyFi contracts: Base, Robinhood (4663) and Arbitrum.
var HookAddresses = []common.Address{
	common.HexToAddress("0xB23F731949145E158E656e1Abe128c5e617A6888"), // base
	common.HexToAddress("0x2AC29f18B22a12917D4653406B0D2Fe7B592A888"), // robinhood
	common.HexToAddress("0xc4f2bE5a31697DCBc7c7FAA0d6BDFaf2b57D2888"), // arbitrum
}

var (
	uScale = uint256.MustFromDecimal("1000000000000000000000000") // HyFi.SCALE = 1e24
	uPips  = uint256.NewInt(pips)

	// slot0 layout: tip 0-39 | ts 40-71 | bookId 72-111 | cur 112-119 | end 120-127 | left 128-223 | ticks 0-3 224-255
	uMask96     = new(uint256.Int).Rsh(new(uint256.Int).SetAllOne(), 160)
	maskCurLeft = new(uint256.Int).Or(new(uint256.Int).Lsh(uint256.NewInt(0xFF), 112),
		new(uint256.Int).Lsh(uMask96, 128))
)
