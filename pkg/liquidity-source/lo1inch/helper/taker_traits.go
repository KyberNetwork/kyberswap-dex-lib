package helper

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Constants for bit positions
const (
	ZX          = "0x"
	UINT_32_MAX = uint64(0xFFFFFFFF)

	makerAmountFlag     = uint(255)
	unwrapWethFlag      = uint(254)
	skipOrderPermitFlag = uint(253)
	usePermit2Flag      = uint(252)
	argsHasReceiver     = uint(251)
	thresholdMaskStart  = uint(0)
	thresholdMaskEnd    = uint(185)
	interactionLenStart = uint(200)
	interactionLenEnd   = uint(224)
	extensionLenStart   = uint(224)
	extensionLenEnd     = uint(248)
)

// AmountMode represents how to treat the amount provided to fill function
type AmountMode int

const (
	// TakerMode - Amount provided to fill function treated as `takingAmount`
	// and `makingAmount` calculated based on it
	TakerMode AmountMode = iota

	// MakerMode - Amount provided to fill function treated as `makingAmount`
	// and `takingAmount` calculated based on it
	MakerMode
)

// Address represents an Ethereum address
type Address string

func NewAddress(addr string) Address {
	return Address(addr)
}

func (a Address) ToString() string {
	return string(a)
}

// / TakerTraits defines the taker's preferences for an order in a single uint256
type TakerTraits struct {
	flags       *big.Int
	receiver    *Address
	extension   *Extension
	interaction *Interaction
}

// EncodeResult represents the encoded result of TakerTraits
type EncodeResult struct {
	TakerTraits *big.Int
	Args        []byte
}

func NewTakerTraits(receiver *Address, ext *Extension, interaction *Interaction) *TakerTraits {
	return &TakerTraits{
		flags:       big.NewInt(0),
		receiver:    receiver,
		extension:   ext,
		interaction: interaction,
	}
}

func DefaultTakerTraits() *TakerTraits {
	return NewTakerTraits(nil, nil, nil)
}

// getBit gets a bit value at the specified position
func (t *TakerTraits) getBit(pos uint) int {
	return int(t.flags.Bit(int(pos)))
}

// setBit sets a bit value at the specified position
func (t *TakerTraits) setBit(pos uint, val int) {
	t.flags.SetBit(t.flags, int(pos), uint(val))
}

func (t *TakerTraits) GetAmountMode() AmountMode {
	return AmountMode(t.getBit(makerAmountFlag))
}

func (t *TakerTraits) SetAmountMode(mode AmountMode) *TakerTraits {
	t.setBit(makerAmountFlag, int(mode))
	return t
}

func (t *TakerTraits) IsNativeUnwrapEnabled() bool {
	return t.getBit(unwrapWethFlag) == 1
}

func (t *TakerTraits) EnableNativeUnwrap() *TakerTraits {
	t.setBit(unwrapWethFlag, 1)
	return t
}

func (t *TakerTraits) DisableNativeUnwrap() *TakerTraits {
	t.setBit(unwrapWethFlag, 0)
	return t
}

func (t *TakerTraits) IsOrderPermitSkipped() bool {
	return t.getBit(skipOrderPermitFlag) == 1
}

func (t *TakerTraits) SkipOrderPermit() *TakerTraits {
	t.setBit(skipOrderPermitFlag, 1)
	return t
}

func (t *TakerTraits) IsPermit2Enabled() bool {
	return t.getBit(usePermit2Flag) == 1
}

func (t *TakerTraits) EnablePermit2() *TakerTraits {
	t.setBit(usePermit2Flag, 1)
	return t
}

func (t *TakerTraits) DisablePermit2() *TakerTraits {
	t.setBit(usePermit2Flag, 0)
	return t
}

func (t *TakerTraits) SetReceiver(receiver Address) *TakerTraits {
	t.receiver = &receiver
	return t
}

func (t *TakerTraits) RemoveReceiver() *TakerTraits {
	t.receiver = nil
	return t
}

func (t *TakerTraits) SetExtension(ext *Extension) *TakerTraits {
	t.extension = ext
	return t
}

func (t *TakerTraits) RemoveExtension() *TakerTraits {
	t.extension = nil
	return t
}

func (t *TakerTraits) SetAmountThreshold(threshold *big.Int) *TakerTraits {
	// Create mask for threshold bits
	mask := new(big.Int).Lsh(big.NewInt(1), thresholdMaskEnd)
	mask.Sub(mask, big.NewInt(1))

	// Clear threshold bits
	t.flags.And(t.flags, mask.Not(mask))

	// Set new threshold
	t.flags.Or(t.flags, threshold)
	return t
}

func (t *TakerTraits) RemoveAmountThreshold() *TakerTraits {
	return t.SetAmountThreshold(big.NewInt(0))
}

func (t *TakerTraits) SetInteraction(interaction *Interaction) *TakerTraits {
	t.interaction = interaction
	return t
}

func (t *TakerTraits) RemoveInteraction() *TakerTraits {
	t.interaction = nil
	return t
}

// Encode encodes the TakerTraits into trait and args
func (t *TakerTraits) Encode() EncodeResult {
	var extensionLen, interactionLen int64

	if t.extension != nil {
		encodedExt := t.extension.Encode()
		if encodedExt != ZX {
			extensionLen = int64(len(strings.TrimPrefix(encodedExt, "0x")) / 2)
		}
	}

	if t.interaction != nil {
		encodedInt := t.interaction.Encode()
		if encodedInt != ZX {
			interactionLen = int64(len(strings.TrimPrefix(encodedInt, "0x")) / 2)
		}
	}

	// Set has receiver bit
	if t.receiver != nil {
		t.setBit(argsHasReceiver, 1)
	} else {
		t.setBit(argsHasReceiver, 0)
	}

	// Set extension length
	extLenMask := new(big.Int).Lsh(big.NewInt(1), extensionLenEnd-extensionLenStart)
	extLenMask.Sub(extLenMask, big.NewInt(1))
	extLen := new(big.Int).SetInt64(extensionLen)
	extLen.Lsh(extLen, extensionLenStart)
	t.flags.Or(t.flags, extLen)

	// Set interaction length
	intLenMask := new(big.Int).Lsh(big.NewInt(1), interactionLenEnd-interactionLenStart)
	intLenMask.Sub(intLenMask, big.NewInt(1))
	intLen := new(big.Int).SetInt64(interactionLen)
	intLen.Lsh(intLen, interactionLenStart)
	t.flags.Or(t.flags, intLen)

	// Build args string
	args := ZX
	if t.receiver != nil {
		args += strings.TrimPrefix(t.receiver.ToString(), ZX)
	}
	if t.extension != nil {
		args += strings.TrimPrefix(t.extension.Encode(), ZX)
	}
	if t.interaction != nil {
		args += strings.TrimPrefix(t.interaction.Encode(), ZX)
	}

	return EncodeResult{
		TakerTraits: t.flags,
		Args:        common.FromHex(args),
	}
}
