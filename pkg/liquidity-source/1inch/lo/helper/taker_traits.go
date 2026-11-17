package helper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AmountMode uint

const (
	makerAmountFlag     = 255
	unwrapWethFlag      = 254
	skipOrderPermitFlag = 253
	usePermit2Flag      = 252
	argsHasReceiver     = 251

	amountThresholdStart    = 0
	amountThresholdEnd      = 185
	argsInteractionLenStart = 200
	argsInteractionLenEnd   = 224
	argsExtensionLenStart   = 224
	argsExtensionLenEnd     = 248

	AmountModeTaker AmountMode = 0
	AmountModeMaker AmountMode = 1
)

//nolint:gochecknoglobals,gomnd,mnd
var (
	// 224-247 bits `ARGS_EXTENSION_LENGTH`   - The length of the extension calldata in the args.
	argsExtensionLenMask = newBitMask(224, 248)
	// 200-223 bits `ARGS_INTERACTION_LENGTH` - The length of the interaction calldata in the args.
	argsInteractionLenMask = newBitMask(200, 224)
	// 0-184 bits                             - The threshold amount.
	amountThresholdMask = newBitMask(0, 185)
)

type TakerTraits struct {
	flags       *big.Int
	receiver    *common.Address
	extension   *Extension
	interaction *Interaction
}

type TakerTraitsOptions struct {
	IsMakingAmount  bool     `json:"is_making_amount"`
	UnwrapWeth      bool     `json:"unwrap_weth"`
	SkipOrderPermit bool     `json:"skip_order_permit"`
	UsePermit2      bool     `json:"use_permit2"`
	Threshold       *big.Int `json:"threshold"`
}

func boolToBit(b bool) uint {
	if b {
		return 1
	}
	return 0
}

func NewTakerTraits(
	flags *big.Int, receiver *common.Address, extension *Extension, interaction *Interaction,
) *TakerTraits {
	return &TakerTraits{
		flags:       flags,
		receiver:    receiver,
		extension:   extension,
		interaction: interaction,
	}
}

func NewDefaultTakerTraits() *TakerTraits {
	return &TakerTraits{
		flags: new(big.Int),
	}
}

func (t *TakerTraits) Decode() TakerTraitsOptions {
	return TakerTraitsOptions{
		IsMakingAmount:  t.IsMakingAmount(),
		UnwrapWeth:      t.UnwrapWeth(),
		SkipOrderPermit: t.SkipOrderPermit(),
		UsePermit2:      t.UsePermit2(),
		Threshold:       t.AmountThreshold(),
	}
}

func (t *TakerTraits) SetAmountMode(mode AmountMode) *TakerTraits {
	t.flags.SetBit(t.flags, makerAmountFlag, uint(mode))
	return t
}

func (t *TakerTraits) IsMakingAmount() bool {
	return t.flags.Bit(makerAmountFlag) != 0
}

func (t *TakerTraits) SetUnwrapWeth(unwrap bool) *TakerTraits {
	t.flags.SetBit(t.flags, unwrapWethFlag, boolToBit(unwrap))
	return t
}

func (t *TakerTraits) UnwrapWeth() bool {
	return t.flags.Bit(unwrapWethFlag) != 0
}

func (t *TakerTraits) SetSkipOrderPermit(skip bool) *TakerTraits {
	t.flags.SetBit(t.flags, skipOrderPermitFlag, boolToBit(skip))
	return t
}

func (t *TakerTraits) SkipOrderPermit() bool {
	return t.flags.Bit(skipOrderPermitFlag) != 0
}

func (t *TakerTraits) SetUsePermit2(use bool) *TakerTraits {
	t.flags.SetBit(t.flags, usePermit2Flag, boolToBit(use))
	return t
}

func (t *TakerTraits) UsePermit2() bool {
	return t.flags.Bit(usePermit2Flag) != 0
}

// SetAmountThreshold sets threshold amount.
// In taker amount mode: the minimum amount a taker agrees to receive in exchange for a taking amount.
// In maker amount mode: the maximum amount a taker agrees to give in exchange for a making amount.
func (t *TakerTraits) SetAmountThreshold(threshold *big.Int) *TakerTraits {
	setMask(t.flags, amountThresholdMask, threshold)
	return t
}

func (t *TakerTraits) AmountThreshold() *big.Int {
	return getMask(t.flags, amountThresholdStart, amountThresholdEnd)
}

// SetExtension sets extension, it is required to provide same extension as in order creation (if any).
func (t *TakerTraits) SetExtension(ext Extension) *TakerTraits {
	t.extension = &ext
	return t
}

// SetInteraction sets interaction, target should implement `ITakerInteraction` interface.
func (t *TakerTraits) SetInteraction(interaction Interaction) *TakerTraits {
	t.interaction = &interaction
	return t
}

func (t *TakerTraits) Encode() (*big.Int, []byte) {
	var extension, interaction []byte
	if t.extension != nil {
		extension = t.extension.Encode()
	}
	if t.interaction != nil {
		interaction = t.interaction.Encode()
	}

	flags := new(big.Int).Set(t.flags)
	if t.receiver != nil {
		flags.SetBit(flags, argsHasReceiver, 1)
	}

	// Set length for extension and interaction.
	setMask(flags, argsExtensionLenMask, big.NewInt(int64(len(extension))))
	setMask(flags, argsInteractionLenMask, big.NewInt(int64(len(interaction))))

	var args []byte
	if t.receiver == nil {
		args = make([]byte, 0, len(extension)+len(interaction))
	} else {
		args = make([]byte, 0, len(t.receiver)+len(extension)+len(interaction))
		args = append(args, t.receiver.Bytes()...)
	}
	args = append(append(args, extension...), interaction...)

	return flags, args
}
