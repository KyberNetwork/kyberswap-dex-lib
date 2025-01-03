package helper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// MakerTraits defines the maker's preferences for an order in a single uint256
// High bits are used for flags:
// 255 bit NO_PARTIAL_FILLS_FLAG          - if set, the order does not allow partial fills
// 254 bit ALLOW_MULTIPLE_FILLS_FLAG      - if set, the order permits multiple fills
// 253 bit                                - unused
// 252 bit PRE_INTERACTION_CALL_FLAG      - if set, the order requires pre-interaction call
// 251 bit POST_INTERACTION_CALL_FLAG     - if set, the order requires post-interaction call
// 250 bit NEED_CHECK_EPOCH_MANAGER_FLAG  - if set, the order requires to check the epoch manager
// 249 bit HAS_EXTENSION_FLAG             - if set, the order has extension(s)
// 248 bit MAKER_USE_PERMIT2_FLAG         - if set, the order uses permit2
// 247 bit MAKER_UNWRAP_WETH_FLAG         - if set, the order requires to unwrap WETH
//
// Low 200 bits are used for allowed sender, expiration, nonceOrEpoch, and series:
// uint80 last 10 bytes of allowed sender address (0 if any)
// uint40 expiration timestamp (0 if none)
// uint40 nonce or epoch
// uint40 series
type MakerTraits struct {
	value *big.Int
}

const (
	// Bit masks for low 200 bits
	allowedSenderStart = uint(0)
	allowedSenderEnd   = uint(80)
	expirationStart    = uint(80)
	expirationEnd      = uint(120)
	nonceOrEpochStart  = uint(120)
	nonceOrEpochEnd    = uint(160)
	seriesStart        = uint(160)
	seriesEnd          = uint(200)

	// Flag bit positions
	noPartialFillsFlag        = uint(255)
	allowMultipleFillsFlag    = uint(254)
	preInteractionCallFlag    = uint(252)
	postInteractionCallFlag   = uint(251)
	needCheckEpochManagerFlag = uint(250)
	hasExtensionFlag          = uint(249)
	makerUsePermit2Flag       = uint(248)
	makerUnwrapWethFlag       = uint(247)
)

// NewMakerTraits creates a new MakerTraits instance with the given value
func NewMakerTraits(val string) *MakerTraits {
	value := new(big.Int)
	if val != "" {
		value.SetString(val, 10)
	}
	return &MakerTraits{value: value}
}

// DefaultMakerTraits returns a MakerTraits instance with default values
func DefaultMakerTraits() *MakerTraits {
	return NewMakerTraits("")
}

// getBit gets a bit value at the specified position
func (mt *MakerTraits) getBit(pos uint) uint {
	return mt.value.Bit(int(pos))
}

// setBit sets a bit value at the specified position
func (mt *MakerTraits) setBit(pos uint, val int) *MakerTraits {
	mt.value.SetBit(mt.value, int(pos), uint(val))
	return mt
}

// getMask gets a value from a bit range
func (mt *MakerTraits) getMask(start, end uint) *big.Int {
	mask := new(big.Int).Lsh(big.NewInt(1), end)
	mask.Sub(mask, new(big.Int).Lsh(big.NewInt(1), start))
	result := new(big.Int).And(mt.value, mask)
	return result.Rsh(result, start)
}

// setMask sets a value in a bit range
func (mt *MakerTraits) setMask(start, end uint, val *big.Int) *MakerTraits {
	// Clear the bits in the range
	clearMask := new(big.Int).Lsh(big.NewInt(1), end)
	clearMask.Sub(clearMask, new(big.Int).Lsh(big.NewInt(1), start))
	clearMask.Not(clearMask)
	mt.value.And(mt.value, clearMask)

	// Set the new value
	valShifted := new(big.Int).Lsh(val, start)
	mt.value.Or(mt.value, valShifted)
	return mt
}

// AllowedSender returns the last 10 bytes of allowed sender address
func (mt *MakerTraits) AllowedSender() common.Address {
	val := mt.getMask(allowedSenderStart, allowedSenderEnd)
	addr := make([]byte, 20)
	val.FillBytes(addr[10:]) // Fill only last 10 bytes
	return common.BytesToAddress(addr)
}

// IsPrivate returns true if the order has a specific allowed sender
func (mt *MakerTraits) IsPrivate() bool {
	return mt.getMask(allowedSenderStart, allowedSenderEnd).Sign() != 0
}

// WithAllowedSender sets the allowed sender for the order
func (mt *MakerTraits) WithAllowedSender(sender common.Address) *MakerTraits {
	if sender == (common.Address{}) {
		return mt.WithAnySender()
	}
	// Take last 10 bytes of the address
	val := new(big.Int).SetBytes(sender.Bytes()[10:])
	return mt.setMask(allowedSenderStart, allowedSenderEnd, val)
}

// WithAnySender removes sender check
func (mt *MakerTraits) WithAnySender() *MakerTraits {
	return mt.setMask(allowedSenderStart, allowedSenderEnd, big.NewInt(0))
}

// Expiration returns the expiration timestamp in seconds, nil if no expiration
func (mt *MakerTraits) Expiration() *big.Int {
	val := mt.getMask(expirationStart, expirationEnd)
	if val.Sign() == 0 {
		return nil
	}
	return val
}

// WithExpiration sets the expiration timestamp
func (mt *MakerTraits) WithExpiration(expiration *big.Int) *MakerTraits {
	if expiration == nil {
		expiration = big.NewInt(0)
	}
	return mt.setMask(expirationStart, expirationEnd, expiration)
}

// NonceOrEpoch returns the nonce or epoch value
func (mt *MakerTraits) NonceOrEpoch() *big.Int {
	return mt.getMask(nonceOrEpochStart, nonceOrEpochEnd)
}

// WithNonce sets the nonce value
func (mt *MakerTraits) WithNonce(nonce *big.Int) *MakerTraits {
	return mt.setMask(nonceOrEpochStart, nonceOrEpochEnd, nonce)
}

// WithEpoch sets the epoch and series values
func (mt *MakerTraits) WithEpoch(series, epoch *big.Int) *MakerTraits {
	mt.setSeries(series)
	mt.enableEpochManagerCheck()
	return mt.WithNonce(epoch)
}

// Series returns the current series value
func (mt *MakerTraits) Series() *big.Int {
	return mt.getMask(seriesStart, seriesEnd)
}

// HasExtension returns true if order has an extension
func (mt *MakerTraits) HasExtension() bool {
	return mt.getBit(hasExtensionFlag) == 1
}

// WithExtension marks that order has an extension
func (mt *MakerTraits) WithExtension() *MakerTraits {
	return mt.setBit(hasExtensionFlag, 1)
}

// IsPartialFillAllowed returns true if partial fills are allowed
func (mt *MakerTraits) IsPartialFillAllowed() bool {
	return mt.getBit(noPartialFillsFlag) == 0
}

// DisablePartialFills disables partial fills for the order
func (mt *MakerTraits) DisablePartialFills() *MakerTraits {
	return mt.setBit(noPartialFillsFlag, 1)
}

// AllowPartialFills allows partial fills for the order
func (mt *MakerTraits) AllowPartialFills() *MakerTraits {
	return mt.setBit(noPartialFillsFlag, 0)
}

// SetPartialFills sets the partial fill flag
func (mt *MakerTraits) SetPartialFills(allowed bool) *MakerTraits {
	if allowed {
		return mt.AllowPartialFills()
	}
	return mt.DisablePartialFills()
}

// IsMultipleFillsAllowed returns true if multiple fills are allowed
func (mt *MakerTraits) IsMultipleFillsAllowed() bool {
	return mt.getBit(allowMultipleFillsFlag) == 1
}

// AllowMultipleFills allows multiple fills for the order
func (mt *MakerTraits) AllowMultipleFills() *MakerTraits {
	return mt.setBit(allowMultipleFillsFlag, 1)
}

// DisableMultipleFills disables multiple fills for the order
func (mt *MakerTraits) DisableMultipleFills() *MakerTraits {
	return mt.setBit(allowMultipleFillsFlag, 0)
}

// SetMultipleFills sets the multiple fills flag
func (mt *MakerTraits) SetMultipleFills(allowed bool) *MakerTraits {
	if allowed {
		return mt.AllowMultipleFills()
	}
	return mt.DisableMultipleFills()
}

// HasPreInteraction returns true if maker has pre-interaction
func (mt *MakerTraits) HasPreInteraction() bool {
	return mt.getBit(preInteractionCallFlag) == 1
}

// EnablePreInteraction enables maker pre-interaction
func (mt *MakerTraits) EnablePreInteraction() *MakerTraits {
	return mt.setBit(preInteractionCallFlag, 1)
}

// DisablePreInteraction disables maker pre-interaction
func (mt *MakerTraits) DisablePreInteraction() *MakerTraits {
	return mt.setBit(preInteractionCallFlag, 0)
}

// HasPostInteraction returns true if maker has post-interaction
func (mt *MakerTraits) HasPostInteraction() bool {
	return mt.getBit(postInteractionCallFlag) == 1
}

// EnablePostInteraction enables maker post-interaction
func (mt *MakerTraits) EnablePostInteraction() *MakerTraits {
	return mt.setBit(postInteractionCallFlag, 1)
}

// DisablePostInteraction disables maker post-interaction
func (mt *MakerTraits) DisablePostInteraction() *MakerTraits {
	return mt.setBit(postInteractionCallFlag, 0)
}

// IsEpochManagerEnabled returns true if epoch manager is enabled
func (mt *MakerTraits) IsEpochManagerEnabled() bool {
	return mt.getBit(needCheckEpochManagerFlag) == 1
}

// IsPermit2 returns true if permit2 is enabled
func (mt *MakerTraits) IsPermit2() bool {
	return mt.getBit(makerUsePermit2Flag) == 1
}

// EnablePermit2 enables permit2 for the order
func (mt *MakerTraits) EnablePermit2() *MakerTraits {
	return mt.setBit(makerUsePermit2Flag, 1)
}

// DisablePermit2 disables permit2 for the order
func (mt *MakerTraits) DisablePermit2() *MakerTraits {
	return mt.setBit(makerUsePermit2Flag, 0)
}

// IsNativeUnwrapEnabled returns true if WETH unwrap is enabled
func (mt *MakerTraits) IsNativeUnwrapEnabled() bool {
	return mt.getBit(makerUnwrapWethFlag) == 1
}

// EnableNativeUnwrap enables WETH unwrap
func (mt *MakerTraits) EnableNativeUnwrap() *MakerTraits {
	return mt.setBit(makerUnwrapWethFlag, 1)
}

// DisableNativeUnwrap disables WETH unwrap
func (mt *MakerTraits) DisableNativeUnwrap() *MakerTraits {
	return mt.setBit(makerUnwrapWethFlag, 0)
}

// Build returns the final traits value
func (mt *MakerTraits) Build() *big.Int {
	return new(big.Int).Set(mt.value)
}

// IsBitInvalidatorMode returns true if bit invalidator mode is used
func (mt *MakerTraits) IsBitInvalidatorMode() bool {
	return !mt.IsPartialFillAllowed() || !mt.IsMultipleFillsAllowed()
}

// IsExpired checks if the order has expired
func (mt *MakerTraits) IsExpired(currentTime int64) bool {
	expiration := mt.Expiration()
	return expiration != nil && expiration.Cmp(big.NewInt(currentTime)) < 0
}

// enableEpochManagerCheck enables epoch manager check
func (mt *MakerTraits) enableEpochManagerCheck() {
	if mt.IsBitInvalidatorMode() {
		panic("Epoch manager allowed only when partialFills and multipleFills enabled")
	}
	mt.setBit(needCheckEpochManagerFlag, 1)
}

// setSeries sets the series value
func (mt *MakerTraits) setSeries(series *big.Int) {
	mt.setMask(seriesStart, seriesEnd, series)
}
