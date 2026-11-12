package oneinchaqua

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// Opcodes of the SwapVM instruction table the simulator runs. The table is shared by both routers; an
// opcode is not in it when the program is not admitted.
const (
	opXYCSwap         = 0x11
	opXYCConcentrate  = 0x12
	opSalt            = 0x14
	opFlatFeeIn       = 0x15
	opAquaProtocolFee = 0x1c
	// opTxOriginGuard is onlyTxOriginTokenBalanceNonZero: strategies carrying it are never pools.
	opTxOriginGuard = 0x21

	feeBpsDenominator = 1_000_000_000

	flatFeeArgsLen     = 4
	protocolFeeArgsLen = 24
	concentrateArgsLen = 64
)

// Instruction is one decoded instruction of the whitelisted subset.
type Instruction struct {
	Op uint8
	// Bps is the fee in 1e9 units, for opFlatFeeIn and opAquaProtocolFee.
	Bps uint32
	// To is the protocol fee recipient, for opAquaProtocolFee.
	To [20]byte
	// SqrtPMin and SqrtPMax are the price bounds of opXYCConcentrate, as 1e18 fixed point.
	SqrtPMin uint256.Int
	SqrtPMax uint256.Int
}

// DiscardReason names why a strategy is not turned into a pool. The values are metric labels.
type DiscardReason string

const (
	ReasonNonCanonical      DiscardReason = "non_canonical"
	ReasonMakerMismatch     DiscardReason = "maker_mismatch"
	ReasonNotAqua           DiscardReason = "not_aqua_order"
	ReasonHooks             DiscardReason = "maker_hooks"
	ReasonReceiver          DiscardReason = "custom_receiver"
	ReasonUnwrap            DiscardReason = "unwrap_weth"
	ReasonAllowZeroAmountIn DiscardReason = "allow_zero_amount_in"
	ReasonSliceIndexes      DiscardReason = "bad_slice_indexes"
	ReasonProgramTruncated  DiscardReason = "program_truncated"
	ReasonProgramEmpty      DiscardReason = "program_empty"
	ReasonTxOriginGuard     DiscardReason = "tx_origin_guard"
	ReasonUnsupportedOpcode DiscardReason = "unsupported_opcode"
	ReasonProgramShape      DiscardReason = "program_shape"
	ReasonBadArgs           DiscardReason = "bad_args"
	ReasonFeeBps            DiscardReason = "fee_bps_out_of_range"
	ReasonPriceBounds       DiscardReason = "bad_price_bounds"
	ReasonFeeRecipient      DiscardReason = "fee_recipient_zero"
	ReasonLaunchFeeRecipent DiscardReason = "launch_zero_fee_recipient"
	ReasonApp               DiscardReason = "unknown_app"
)

// AdmissionError carries the discard reason of a refused strategy.
type AdmissionError struct {
	Reason DiscardReason
	Detail string
}

func (e *AdmissionError) Error() string {
	if e.Detail == "" {
		return "1inch-aqua: strategy refused: " + string(e.Reason)
	}
	return fmt.Sprintf("1inch-aqua: strategy refused: %s (%s)", e.Reason, e.Detail)
}

func refuse(reason DiscardReason, detail string, args ...any) error {
	return &AdmissionError{Reason: reason, Detail: fmt.Sprintf(detail, args...)}
}

// ReasonOf returns the discard reason of an admission error.
func ReasonOf(err error) DiscardReason {
	if ae, ok := err.(*AdmissionError); ok {
		return ae.Reason
	}
	return ""
}

// ParseOrder checks the order against the admission rules that need only the order itself (maker
// traits and the program) and returns the decoded program. launch selects the router whose fee rules
// apply.
func ParseOrder(o *Order, launch bool) ([]Instruction, error) {
	if !traitsBit(&o.Traits, traitsUseAqua) {
		return nil, refuse(ReasonNotAqua, "signature order")
	}
	if traitsBit(&o.Traits, traitsShouldUnwrap) {
		return nil, refuse(ReasonUnwrap, "")
	}
	if traitsBit(&o.Traits, traitsAllowZeroAmountIn) {
		return nil, refuse(ReasonAllowZeroAmountIn, "")
	}
	var flags uint256.Int
	flags.Rsh(&o.Traits, traitsHookFlagsShift)
	if flags.Uint64()&traitsHookFlagsMask != 0 {
		return nil, refuse(ReasonHooks, "")
	}
	var receiver common.Address
	receiverWord := o.Traits.Bytes32()
	copy(receiver[:], receiverWord[12:])
	if receiver != (common.Address{}) && receiver != o.Maker {
		return nil, refuse(ReasonReceiver, "")
	}
	// With no hook flag set the four slice ends must all be zero, so the program is the whole data.
	for n := range uint(4) {
		if idx := sliceIndex(&o.Traits, n); idx != 0 {
			return nil, refuse(ReasonSliceIndexes, "index%d=%d, data length %d", n, idx, len(o.Data))
		}
	}
	return parseProgram(o.Data, launch)
}

// parseProgram decodes [op][argsLen][args] repeated to the end and enforces the whitelist.
func parseProgram(program []byte, launch bool) ([]Instruction, error) {
	if len(program) == 0 {
		return nil, refuse(ReasonProgramEmpty, "")
	}
	var (
		out        = make([]Instruction, 0, 6)
		sawGuard   bool
		firstOther DiscardReason
		otherInfo  string
	)
	note := func(r DiscardReason, detail string) {
		if firstOther == "" {
			firstOther, otherInfo = r, detail
		}
	}
	for pc := 0; pc < len(program); {
		if pc+2 > len(program) {
			return nil, finishWalk(sawGuard, refuse(ReasonProgramTruncated, "missing header at %d", pc))
		}
		op, argsLen := program[pc], int(program[pc+1])
		pc += 2
		if pc+argsLen > len(program) {
			return nil, finishWalk(sawGuard, refuse(ReasonProgramTruncated, "args of opcode %#x overrun", op))
		}
		args := program[pc : pc+argsLen]
		pc += argsLen

		switch op {
		case opTxOriginGuard:
			sawGuard = true
		case opSalt:
			out = append(out, Instruction{Op: op})
		case opXYCSwap:
			if argsLen != 0 {
				note(ReasonBadArgs, "xycSwap takes no args")
			}
			out = append(out, Instruction{Op: op})
		case opFlatFeeIn:
			if argsLen != flatFeeArgsLen {
				note(ReasonBadArgs, "flatFeeIn args length")
				break
			}
			ins := Instruction{Op: op, Bps: uint32(args[0])<<24 | uint32(args[1])<<16 | uint32(args[2])<<8 | uint32(args[3])}
			if ins.Bps > feeBpsDenominator {
				note(ReasonFeeBps, "flatFeeIn bps")
			}
			out = append(out, ins)
		case opAquaProtocolFee:
			if argsLen != protocolFeeArgsLen {
				note(ReasonBadArgs, "aquaProtocolFee args length")
				break
			}
			ins := Instruction{Op: op, Bps: uint32(args[0])<<24 | uint32(args[1])<<16 | uint32(args[2])<<8 | uint32(args[3])}
			copy(ins.To[:], args[4:])
			switch {
			case ins.Bps > feeBpsDenominator:
				note(ReasonFeeBps, "aquaProtocolFee bps")
			case ins.To == [20]byte{} && launch:
				note(ReasonLaunchFeeRecipent, "")
			case ins.To == [20]byte{} && ins.Bps != 0:
				note(ReasonFeeRecipient, "")
			}
			out = append(out, ins)
		case opXYCConcentrate:
			if argsLen != concentrateArgsLen {
				note(ReasonBadArgs, "xycConcentrate args length")
				break
			}
			ins := Instruction{Op: op}
			ins.SqrtPMin.SetBytes32(args[:32])
			ins.SqrtPMax.SetBytes32(args[32:])
			if !ins.SqrtPMin.Lt(&ins.SqrtPMax) {
				note(ReasonPriceBounds, "sqrtPmin >= sqrtPmax")
			}
			out = append(out, ins)
		default:
			note(ReasonUnsupportedOpcode, fmt.Sprintf("opcode %#x", op))
		}
	}
	if sawGuard {
		return nil, refuse(ReasonTxOriginGuard, "")
	}
	if firstOther != "" {
		return nil, refuse(firstOther, "%s", otherInfo)
	}
	if err := checkShape(out); err != nil {
		return nil, err
	}
	return out, nil
}

// finishWalk lets the tx.origin guard win over a malformed tail: the strategy is excluded either way,
// but the guard is the reason 1inch asked for.
func finishWalk(sawGuard bool, err error) error {
	if sawGuard {
		return refuse(ReasonTxOriginGuard, "")
	}
	return err
}

// checkShape accepts only programs the interpreter runs without a router precondition failing: exactly
// one xycSwap, every fee and concentrate instruction before it, only salt after it.
func checkShape(prog []Instruction) error {
	swaps := 0
	for _, ins := range prog {
		switch ins.Op {
		case opXYCSwap:
			swaps++
		case opSalt:
		default:
			if swaps > 0 {
				return refuse(ReasonProgramShape, "wrapper after xycSwap")
			}
		}
	}
	if swaps != 1 {
		return refuse(ReasonProgramShape, "%d xycSwap instructions", swaps)
	}
	return nil
}

// Maker traits layout of the SwapVM order.
const (
	traitsShouldUnwrap       = 255
	traitsUseAqua            = 254
	traitsAllowZeroAmountIn  = 253
	traitsHookFlagsMask      = 0xf // bits 249..252: the four transfer hooks
	traitsHookFlagsShift     = 249
	traitsSliceIndexesOffset = 160
	traitsSliceIndexBits     = 16
)

// Order is the SwapVM order the router receives: abi.encode(Order) of these three fields is both the
// bytes a maker ships to Aqua and the preimage of the strategy hash.
type Order struct {
	Maker  common.Address
	Traits uint256.Int
	Data   []byte
}

// Encode returns abi.encode(order) for the tuple (address maker, uint256 traits, bytes data).
func (o *Order) Encode() []byte {
	padded := (len(o.Data) + 31) &^ 31
	out := make([]byte, 32*5+padded)
	out[31] = 0x20 // offset of the tuple
	copy(out[32+12:], o.Maker[:])
	o.Traits.WriteToSlice(out[64:96])
	out[96+31] = 0x60 // offset of data inside the tuple
	new(uint256.Int).SetUint64(uint64(len(o.Data))).WriteToSlice(out[128:160])
	copy(out[160:], o.Data)
	return out
}

// Hash is keccak256(abi.encode(order)), the key under which the router reads Aqua balances.
func (o *Order) Hash() common.Hash {
	return crypto.Keccak256Hash(o.Encode())
}

// DecodeOrder parses the strategy bytes of a Shipped event as abi.encode(Order). Only the canonical
// encoding is accepted (same offsets, zero padding), because the strategy hash is taken over exactly
// these bytes: a loose decoder would admit a strategy whose router key differs from its Aqua key.
func DecodeOrder(raw []byte) (*Order, error) {
	if len(raw) < 32*5 {
		return nil, ErrNonCanonical
	}
	word := func(i int) []byte { return raw[32*i : 32*(i+1)] }
	var (
		zero12 [12]byte
		o      Order
	)
	if !isWord(word(0), 0x20) || !isWord(word(3), 0x60) || !bytes.Equal(word(1)[:12], zero12[:]) {
		return nil, ErrNonCanonical
	}
	copy(o.Maker[:], word(1)[12:])
	o.Traits.SetBytes32(word(2))
	var length uint256.Int
	length.SetBytes32(word(4))
	if !length.IsUint64() || length.Uint64() > uint64(len(raw)) {
		return nil, ErrNonCanonical
	}
	n := int(length.Uint64())
	if len(raw) != 32*5+((n+31)&^31) {
		return nil, ErrNonCanonical
	}
	o.Data = bytes.Clone(raw[160 : 160+n])
	if !bytes.Equal(o.Encode(), raw) {
		return nil, ErrNonCanonical
	}
	return &o, nil
}

func isWord(w []byte, v byte) bool {
	for _, b := range w[:31] {
		if b != 0 {
			return false
		}
	}
	return w[31] == v
}

// traitsBit reports whether bit i of the maker traits is set.
func traitsBit(traits *uint256.Int, i uint) bool {
	return traits[i/64]>>(i%64)&1 == 1
}

// sliceIndex returns the cumulative end offset of order data slice n (0..3) from the maker traits.
func sliceIndex(traits *uint256.Int, n uint) uint64 {
	bit := traitsSliceIndexesOffset + n*traitsSliceIndexBits
	var shifted uint256.Int
	shifted.Rsh(traits, bit)
	return shifted.Uint64() & 0xffff
}

// parseUint256Hex reads a 0x hex word, with or without leading zeros.
func parseUint256Hex(s string) (*uint256.Int, error) {
	digits := strings.TrimLeft(strings.TrimPrefix(s, "0x"), "0")
	if digits == "" {
		return new(uint256.Int), nil
	}
	return uint256.FromHex("0x" + digits)
}
