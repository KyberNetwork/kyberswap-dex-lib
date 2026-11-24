package efficientprop

import (
	"encoding/binary"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// The layout of a desk contract (the builder code of Ethereum and BNB Chain). Slot 1 is the desk word: [0-159] the
// pusher, [160-190] the desk epoch D, [224] stopped, [225-255] the block of the stop; slot 2 the registry and slot 3
// the word key. Only the front's ops write slots 1-3. Pair slot i (the listing's word index, 0-255) has a quote word
// at quoteWordBase + i, which has the stamp at bits 0-31, and a firm word at firmWordBase + i, which counts only where
// the quote word's bit 205 is set.
const (
	deskWordSlot  = 1
	quoteWordBase = 0x100
	firmWordBase  = 0x200

	// the desk word's bits (the desk contract's slot 1, not the front's deskOf record, whose stop sits at bit 192)
	deskEpochLo, deskEpochBits = 160, 31
	deskStoppedBit             = 224
	// the quote word's bits: the desk epoch it was written under, and the firm state
	quoteEpochLo, quoteEpochBits = 32, 31
	quoteFirmBit                 = 205
)

// binding is where a listing sits in its front: its id and, from its doorView, the desk it is bound to (0: none) and
// its pair slot in that desk's contract.
type binding struct {
	listing   uint32
	desk      uint32
	wordIndex uint8
}

// listingSlots is where a listing's desk update sits in a /state answer under the per-desk rule: the desk contract
// the update writes; the listing's own slots in it (its quote word and its firm word); the quote word's slot, which
// holds the update's block stamp, the low 32 bits of the block the update is for; and the firm word's slot, which is
// the listing's own only where its quote word has the firm bit set. Several listings may share one desk contract, each
// with slots of its own.
//
// The desk word's slot is read at the chain's head, never from an answer, into deskWord (deskKnown once it is). The
// listing's quote word counts only while the desk is not stopped and the word's desk epoch is the desk word's, all 31
// bits (fresh). An answer that writes any slot of the contract but a pair's quote or firm word, whichever pair, covers
// none of the contract's listings (foreign). The head's own quote word of the listing is read as well (headQuote,
// headKnown), for an update stamped with the block before the target's and for a firm quote the answer gives no firm
// word.
type listingSlots struct {
	account common.Address
	slots   []common.Hash
	stampAt common.Hash
	firmAt  common.Hash

	deskWord  common.Hash
	deskKnown bool

	headQuote common.Hash
	headKnown bool

	headFirm      common.Hash
	headFirmKnown bool
}

// owns is whether slot is one of the listing's own, given the quote word the answer holds for it: its quote word, or
// its firm word under the quote word's firm bit.
func (l listingSlots) owns(slot common.Hash, quote common.Hash) bool {
	if slot == l.stampAt {
		return true
	}
	if slot == l.firmAt {
		return firm(quote)
	}
	return false
}

// firm is whether a quote word has the firm bit.
func firm(quote common.Hash) bool {
	return new(big.Int).SetBytes(quote[:]).Bit(quoteFirmBit) == 1
}

// foreign is whether slot is one no answer may write on the desk contract: every slot but a pair's quote or firm word.
func (l listingSlots) foreign(slot common.Hash) bool {
	n := new(big.Int).SetBytes(slot[:])
	return !inPairs(n, quoteWordBase) && !inPairs(n, firmWordBase)
}

// inPairs is whether n is base + i for an i of 0-255.
func inPairs(n *big.Int, base int64) bool {
	return n.Cmp(big.NewInt(base)) >= 0 && n.Cmp(big.NewInt(base+255)) <= 0
}

// fresh is whether the quote word w counts under the desk word read at the head: the desk is not stopped and w's desk
// epoch is the desk word's. A desk word not read counts no word.
func (l listingSlots) fresh(w common.Hash) bool {
	if !l.deskKnown {
		return false
	}
	d, q := new(big.Int).SetBytes(l.deskWord[:]), new(big.Int).SetBytes(w[:])
	return d.Bit(deskStoppedBit) == 0 && bits(d, deskEpochLo, deskEpochBits) == bits(q, quoteEpochLo, quoteEpochBits)
}

// same is whether two reads name the same slots: the same desk contract and the same quote and firm words.
func (l listingSlots) same(o listingSlots) bool {
	return l.account == o.account && l.stampAt == o.stampAt && l.firmAt == o.firmAt && slices.Equal(l.slots, o.slots)
}

// deskRule is the slot rule of the builder code (Ethereum and BNB Chain), where a maker's desk contract holds its
// several listings, a pair slot each. A listing's account is its desk's contract: deskCellOf(desk), which must be the
// CREATE2 address of the front at salt (1 << 252) | desk under cellHash, the front's CELL_CODEHASH. Its own slots are
// its quote word and its firm word, and it is guarded by the desk word at slot 1.
type deskRule struct {
	cellHash common.Hash
}

// deskContractOf is desk's contract: the CREATE2 address of front at salt (1 << 252) | desk under cellHash.
func deskContractOf(front common.Address, cellHash common.Hash, desk uint32) common.Address {
	salt := new(big.Int).Lsh(big.NewInt(1), 252)
	salt.Or(salt, new(big.Int).SetUint64(uint64(desk)))
	return crypto.CreateAddress2(front, common.BigToHash(salt), cellHash[:])
}

func pairSlot(base int64, wordIndex uint8) common.Hash {
	return common.BigToHash(big.NewInt(base + int64(wordIndex)))
}

// read is the call that names the desk contract of front's listing, bound to a desk (never 0), and a new holder for
// its answer, which slots reads once the answer is unpacked into it.
func (deskRule) read(front common.Address, b binding) (call, any) {
	return call{abi: &frontABI, target: front, method: "deskCellOf", params: []any{b.desk}}, new(common.Address)
}

// slots is the listing's slots as read's answer, unpacked into holder, names them; false where it names none or names
// an account that is not the desk's CREATE2 child.
func (r deskRule) slots(front common.Address, b binding, holder any) (listingSlots, bool) {
	cell, ok := holder.(*common.Address)
	if !ok || cell == nil || *cell == (common.Address{}) || b.desk == 0 ||
		*cell != deskContractOf(front, r.cellHash, b.desk) {
		return listingSlots{}, false
	}
	quote, firm := pairSlot(quoteWordBase, b.wordIndex), pairSlot(firmWordBase, b.wordIndex)
	return listingSlots{account: *cell, slots: []common.Hash{quote, firm}, stampAt: quote, firmAt: firm}, true
}

// deskWord is the account and slot the head's desk word is read from for a listing bound to desk.
func (r deskRule) deskWord(front common.Address, desk uint32) (common.Address, common.Hash, bool) {
	if desk == 0 {
		return common.Address{}, common.Hash{}, false
	}
	return deskContractOf(front, r.cellHash, desk), common.BigToHash(big.NewInt(deskWordSlot)), true
}

// lowStamp is a slot's low 32 bits.
func lowStamp(w common.Hash) uint32 { return binary.BigEndian.Uint32(w[common.HashLength-4:]) }

// slotsOf is listing b's slots as rule names them from ret, the return data of its read c (nil where it reverted),
// unpacked into holder; false where it names none.
func slotsOf(rule deskRule, front common.Address, b binding, c call, holder any, ret []byte) (listingSlots, bool) {
	if ret == nil || c.abi.UnpackIntoInterface(holder, c.method, ret) != nil {
		return listingSlots{}, false
	}
	return rule.slots(front, b, holder)
}
