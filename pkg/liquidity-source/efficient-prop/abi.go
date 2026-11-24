package efficientprop

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// The ABIs are parsed as package variables, not in init, so the event topics derived from them (event_parser.go)
// are set when those variables are.
var (
	doorABI     = mustABI(doorJSON)
	frontABI    = mustABI(frontJSON)
	factoryABI  = mustABI(factoryJSON)
	cellABI     = mustABI(cellJSON)
	baseTimeABI = mustABI([]byte(`[{"type":"function","name":"timestampMs","stateMutability":"view","inputs":[],` +
		`"outputs":[{"name":"","type":"uint256"}]}]`))
	// laneABI is a stored price lane's hold, how long it holds a replaced print on the fronts' clock, ms, and its
	// post's log: the number of feeds a post stored, none where it was a no-op
	laneABI = mustABI([]byte(`[{"type":"function","name":"HOLD_MS","stateMutability":"view","inputs":[],` +
		`"outputs":[{"name":"","type":"uint256"}]},{"type":"event","name":"Posted","anonymous":false,` +
		`"inputs":[{"name":"stored","type":"uint256","indexed":false}]}]`))
)

func mustABI(j []byte) abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader(j))
	if err != nil {
		panic(err)
	}
	return parsed
}

// DoorView is the front's cold read of one listing, field for field its `doorView(uint24)` answer: all zero for a
// listing never reserved.
type DoorView struct {
	PairWord    *big.Int       `json:"pairWord"`
	Token0      common.Address `json:"token0"`
	Token1      common.Address `json:"token1"`
	Pool        common.Address `json:"pool"`
	BaseIndex   uint8          `json:"baseIndex"`
	QuoteIndex  uint8          `json:"quoteIndex"`
	Flags       uint8          `json:"flags"`
	KeyEpoch    uint32         `json:"keyEpoch"`
	Desk        uint32         `json:"desk"`
	WordIndex   uint8          `json:"wordIndex"`
	Scopes      uint8          `json:"scopes"`
	FenceBidPpm uint32         `json:"fenceBidPpm"`
	FenceAskPpm uint32         `json:"fenceAskPpm"`
	TipCap      uint16         `json:"tipCap"`
	ConfMul     uint8          `json:"confMul"`
	CreditMode  uint8          `json:"creditMode"`
	BidFillMs   *big.Int       `json:"bidFillMs"`
	AskFillMs   *big.Int       `json:"askFillMs"`
	Ticket      *big.Int       `json:"ticket"`
	Word        *big.Int       `json:"word"`
	Depth       [4]*big.Int    `json:"depth"`
}

// doorViewOut is the shape ethrpc unpacks doorView's one tuple output into.
type doorViewOut struct {
	View DoorView
}

// tipCapWei is a tip cap as the front stores it, a 12-bit mantissa times 10 to its 4-bit exponent, in wei per gas
// above the base fee. Zero on a storage build: no word stands, and the listing fills nothing; on any other: no cap.
func tipCapWei(c uint16) uint64 {
	v := uint64(c & 0xfff)
	for range (c >> 12) & 15 {
		v *= 10
	}
	return v
}

// fillableOut is a door's quoteFillable answer: the prefix of the amount asked that the listing fills now, and what
// it pays.
type fillableOut struct {
	FillableAmountIn *big.Int
	AmountOut        *big.Int
}

// tiedScopes is a tied listing's scopes bits 1-4 (the binding's fence kind and carry life read as one number): its
// fills walk its desk's levels from its print, so their age runs from the print's time. It is not quoted.
const tiedScopes = 8

// entryGuardBit is a source entry's guard (E_GUARD): a fill reading a guarded source asks its lane for the fill's side
// over the prints the lane holds, so a print the lane replaced still prices that side until the lane's hold (HOLD_MS)
// after the post that replaced it has passed, and the side is priced on the kept print alone from then on.
const entryGuardBit = 247

// kindSource is the change kind the front's SettingsSet logs for a listing's source change (setSource): the source
// ids of its record word 10.
const kindSource = 34

// frontClock is the front's clock() answer, seven words: its maker-time windows and clock, ms. On a block-count clock
// pin then records how it reads at the state's block, which sets where place puts a later moment.
type frontClock struct {
	MaxLeadMs  *big.Int
	MaxTtlMs   *big.Int
	PullLeadMs *big.Int
	AgeSlackMs *big.Int
	ClockMode  *big.Int
	ClockBlock *big.Int
	ClockMs    *big.Int

	// reads is how a block-count clock reads at the state's block (pin): readsCount, readsBaseTime or readsSecond
	reads uint8
	// baseTime is the chain's millisecond-time contract (Front.BaseTime), which a block-count clock reads
	baseTime common.Address
	// number0 and baseTime0 are the state's block and BaseTime's reading in it, ms, where the clock reads BaseTime
	number0, baseTime0 uint64
}

// How a block-count clock reads at the state's block (frontClock.pin): its count, which place follows (the zero value,
// so a clock never pinned places on its count); BaseTime, a second or more off the count, which place follows by state
// overrides; or neither, the start of the block's second, which no override places.
const (
	readsCount uint8 = iota
	readsBaseTime
	readsSecond
)

// The front's clock modes (clock().clockMode) an override can place: mode 0 reads the block's second, mode 2 the
// block number's milliseconds from its anchor, blockMs a block, where they agree with the block's second.
const (
	clockSeconds = 0
	clockBlocks  = 2
	blockMs      = 200
)

// place is the block override in which the front's clock reads the latest moment, ms, no later than ms that a block
// may read: on a seconds clock (mode 0) the second alone, the age clock then its second plus the age slack. On a
// block-count clock (mode 2) it follows the clock as it read at the state's block (pin). Where it read its count, the
// second and the block number whose milliseconds from the anchor are that moment, so the clock reads them, where the
// second alone would leave it the pinned block's sub-second. Where it read BaseTime, a second or more off the count,
// the moment itself: its second, the chain's block by 200 ms from BaseTime's reading at the state's block, and
// BaseTime's stored sub-second set to the moment's by a state override (baseTimeMs), so a moment late in its second,
// its last included, is placed exactly, older than any fill of that second; ok only where the front's clock reads
// that moment there. ok is false where it read neither (the start of its second), and on any other clock (BaseTime's
// mode 1 or a precompile's), whose sub-second this source does not set.
func (c frontClock) place(ms uint64) (o override, ok bool) {
	switch c.ClockMode.Uint64() {
	case clockSeconds:
		return override{time: ms / 1000}, ms >= 1000
	case clockBlocks:
		switch c.reads {
		case readsSecond:
			return o, false
		case readsBaseTime:
			if ms < c.baseTime0 {
				return o, false
			}
			o = override{time: ms / 1000, number: c.number0 + (ms-c.baseTime0)/blockMs, baseTimeMs: ms,
				baseTime: c.baseTime}
			return o, c.moment(o.time, o.number, new(big.Int).SetUint64(ms)) == ms
		}
		ms0, block0 := c.ClockMs.Uint64(), c.ClockBlock.Uint64()
		if ms < ms0 || block0 == 0 {
			return o, false
		}
		k := (ms - ms0) / blockMs
		return override{time: (ms0 + k*blockMs) / 1000, number: block0 + k}, true
	}
	return o, false
}

// firstFrom is the block override in which the front's clock reads the earliest moment at or after ms that a block may
// read, where place puts it: on a seconds clock the first second at or after it; on a block-count clock that read its
// count, the first 200 ms block whose count is at or after it; where it read BaseTime, the moment itself (place), the
// earliest a block may store. ok is false where place would refuse that moment.
func (c frontClock) firstFrom(ms uint64) (o override, ok bool) {
	switch c.ClockMode.Uint64() {
	case clockSeconds:
		return c.place((ms + 999) / 1000 * 1000)
	case clockBlocks:
		if c.reads == readsCount {
			if ms0 := c.ClockMs.Uint64(); ms > ms0 {
				ms = ms0 + (ms-ms0+blockMs-1)/blockMs*blockMs
			}
		}
		return c.place(ms)
	}
	return o, false
}

// moment is the front's clock in a block of unix second sec and number, ms, as PropAMMClock.read reads it: on a
// block-count clock the anchor's milliseconds where they fall in the block's second, else BaseTime's (baseTimeMs,
// nil where it did not answer) where they do, else the start of the second; on any other clock the start of the
// second, the least it may read.
func (c frontClock) moment(sec, number uint64, baseTimeMs *big.Int) uint64 {
	if c.ClockMode.Uint64() == clockBlocks {
		if ms0, block0 := c.ClockMs.Uint64(), c.ClockBlock.Uint64(); number >= block0 {
			if ms := ms0 + (number-block0)*blockMs; ms/1000 == sec {
				return ms
			}
		}
		if baseTimeMs != nil && baseTimeMs.IsUint64() && baseTimeMs.Uint64()/1000 == sec {
			return baseTimeMs.Uint64()
		}
	}
	return sec * 1000
}

// pin records how a block-count clock reads at the state's block, of unix second sec and number, where BaseTime
// answered baseTimeMs (nil: no answer), as PropAMMClock.read reads it, for place; stepsBack is whether the clock may
// read earlier at a later block. On Base the count sits on BaseTime's reading, and a block cadence other than 200 ms
// after the anchor moves it off: off by less than a second, the front reads the count in the blocks where it falls in
// the block's second and BaseTime in the others, so it may step back; a second or more off, it reads BaseTime in every
// block, and the count's blocks are not where it reads. Any other clock is not pinned and never steps back here.
func (c *frontClock) pin(sec, number uint64, baseTimeMs *big.Int) (stepsBack bool) {
	if c.ClockMode.Uint64() != clockBlocks {
		return false
	}
	ms0, block0 := c.ClockMs.Uint64(), c.ClockBlock.Uint64()
	counted := number >= block0
	count := ms0 + (number-block0)*blockMs
	bt, btOK := uint64(0), baseTimeMs != nil && baseTimeMs.IsUint64() && baseTimeMs.Uint64()/1000 == sec
	if btOK {
		bt = baseTimeMs.Uint64()
	}
	switch {
	case counted && btOK && count != bt && max(count, bt)-min(count, bt) < 1000:
		return true
	case counted && count/1000 == sec:
		c.reads = readsCount
	case btOK:
		c.reads, c.number0, c.baseTime0 = readsBaseTime, number, bt
	default:
		c.reads = readsSecond
	}
	return false
}

// valid is whether every word of the clock was read and fits 64 bits.
func (c frontClock) valid() bool {
	for _, v := range []*big.Int{c.MaxTtlMs, c.AgeSlackMs, c.ClockMode, c.ClockBlock, c.ClockMs} {
		if v == nil || !v.IsUint64() {
			return false
		}
	}
	return true
}

// bits is the n-bit field of w from bit lo up.
func bits(w *big.Int, lo, n uint) uint64 {
	if w == nil {
		return 0
	}
	f := new(big.Int).Rsh(w, lo)
	return f.And(f, new(big.Int).SetUint64(1<<n-1)).Uint64()
}

// validUntil is the last unix second in which a fill of the listing certainly finds its desk's word standing and the
// depth of each direction in quoted (0: token0 in, 1: token1 in) unexpired; 0 where it has no word. The front fills
// from a word while its age clock reads at most the word's maker time (bits 72-119, ms) plus its life (bits
// 120-135, s), capped at maxTtlMs. Every clock mode reads a block's own second: the block's milliseconds where it
// knows them, else the start of the second plus ageSlackMs, so in a block of unix second T the age clock reads at
// most T*1000 + max(999, ageSlackMs). A side's depth word (bid: depth[0], ask: depth[2]) fills while the block's
// unix second is under its expiry (bits 60-91, s).
func validUntil(v DoorView, maxTtlMs, ageSlackMs uint64, quoted [2]bool) uint64 {
	if v.Word == nil || v.Word.Sign() == 0 {
		return 0
	}
	life := min(bits(v.Word, 120, 16)*1000, maxTtlMs)
	end := bits(v.Word, 72, 48) + life
	lead := max(999, ageSlackMs)
	if end < lead {
		return 0
	}
	until := (end - lead) / 1000
	// the pair word's bit 8: the pair's base is token 0, and a fill selling the base is the bid's
	baseIsToken0 := v.PairWord != nil && v.PairWord.Bit(8) == 1
	for dir, q := range quoted {
		if !q {
			continue
		}
		d := v.Depth[2]
		if (dir == 0) == baseIsToken0 {
			d = v.Depth[0]
		}
		expires := bits(d, 60, 32)
		if expires == 0 {
			return 0
		}
		until = min(until, expires-1)
	}
	return until
}

// frontStore is the front's storage namespace (its Layout's slot 0), frontSourcesIndex the Layout's slot of its price
// sources' entries (mapping(uint256 => uint256)), and frontPoolsIndex that of its listing records (mapping(uint256 =>
// Pool)), whose word liveSourceWord is the listing's live source, word controlsWord its controls word, word
// sourceIdsWordIndex its bound sources and word sessionWordIndex its session word.
var frontStore = common.HexToHash("0xca023b73b2185d1ba92ba3c09f7870c56c9bdadd91407b68ebb408947da60b00")

const (
	frontSourcesIndex  = 15
	frontPoolsIndex    = 18
	liveSourceWord     = 2
	controlsWord       = 7
	sourceIdsWordIndex = 10
	sessionWordIndex   = 13
)

// liveSourceSlot is the front's storage slot of a listing's live source word, its record's word 2: the copy of the
// source it reads now (its primary's, or its backup's once the primary is darkened) that a fill reads alone, its guard
// (E_GUARD) at bit 247 as the entry holds it; zero while the listing has no source.
func liveSourceSlot(listing uint32) common.Hash {
	return listingSlot(listing, liveSourceWord)
}

// controlsSlot is the front's storage slot of a listing's controls word: its record at
// keccak256(abi.encode(listing, namespace + 18)), word 7.
func controlsSlot(listing uint32) common.Hash {
	return listingSlot(listing, controlsWord)
}

// sessionSlot is the front's storage slot of a listing's session word, its record's word 13: a SESSION listing's
// closed-phase settings, zero for none, its closedAfter (s) in bits 0-15.
func sessionSlot(listing uint32) common.Hash {
	return listingSlot(listing, sessionWordIndex)
}

// implementationSlot is a proxy's ERC-1967 implementation slot, keccak256("eip1967.proxy.implementation") - 1: the
// front's implementation is its low 20 bytes.
var implementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

// sourceIdsSlot is the front's storage slot of a listing's bound source ids, its record's word 10: the primary at bits
// 0-31, the backup at 32-63 (0: none); zero while the listing has no source.
func sourceIdsSlot(listing uint32) common.Hash {
	return listingSlot(listing, sourceIdsWordIndex)
}

// sourceSlot is the front's storage slot of a price source's entry, written once by registerSource and never edited:
// keccak256(abi.encode(id, namespace + 15)). Bits 0-159 of the entry are the contract a fill calls for the source (a
// storage build's stored price lane, a builder build's pull check).
func sourceSlot(id uint32) common.Hash {
	base := new(big.Int).Add(frontStore.Big(), big.NewInt(frontSourcesIndex))
	return crypto.Keccak256Hash(common.BigToHash(big.NewInt(int64(id))).Bytes(), common.BigToHash(base).Bytes())
}

// listingSlot is the front's storage slot of word w of a listing's record.
func listingSlot(listing uint32, w int64) common.Hash {
	base := new(big.Int).Add(frontStore.Big(), big.NewInt(frontPoolsIndex))
	rec := crypto.Keccak256(common.BigToHash(big.NewInt(int64(listing))).Bytes(), common.BigToHash(base).Bytes())
	return common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(rec), big.NewInt(w)))
}
