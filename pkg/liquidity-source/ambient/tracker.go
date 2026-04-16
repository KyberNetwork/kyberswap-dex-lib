package ambient

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TrackedLevel struct {
	Tick  int32
	Level BookLevel
}

type TrackedKnockout struct {
	Tick      int32
	BidPivot  KnockoutPivot
	BidMerkle KnockoutMerkle
	AskPivot  KnockoutPivot
	AskMerkle KnockoutMerkle
}

// TrackerExtra is the research-side snapshot emitted by PoolTracker.
type TrackerExtra struct {
	Base           common.Address
	Quote          common.Address
	PoolIdx        uint64
	PoolHash       common.Hash
	Curve          CurveState
	PoolSpec       PoolSpec
	TemplateSpec   PoolSpec
	PoolParams     PoolParams
	TemplateParams PoolParams
	ActiveTicks    []int32
	Levels         []TrackedLevel
	Knockouts      []TrackedKnockout
}

type StateTracker struct {
	Client  *ethclient.Client
	DexAddr common.Address
	Fetcher *TickFetcher
}

func NewStateTracker(c *ethclient.Client, dexAddr common.Address) *StateTracker {
	return &StateTracker{
		Client:  c,
		DexAddr: dexAddr,
		Fetcher: NewTickFetcher(c, dexAddr),
	}
}

func (p PoolSpec) ToPoolParams() PoolParams {
	return PoolParams{
		FeeRate:      p.FeeRate,
		ProtocolTake: p.ProtocolTake,
		TickSize:     p.TickSize,
	}
}

// Load fetches a complete pool snapshot using JSON-RPC batching. The lobby
// sweep, terminus reads, and per-tick reads are each issued as a single
// batched HTTP roundtrip — 3 RTTs total instead of ~270 sequential calls.
func (t *StateTracker) Load(
	ctx context.Context,
	base,
	quote common.Address,
	poolIdx uint64,
	blockNum *big.Int,
) (*TrackerExtra, error) {
	orderedBase, orderedQuote := normalizePair(base, quote)
	poolHash := EncodePoolHash(orderedBase, orderedQuote, poolIdx)

	// ---- Stage A: curve(2) + poolSpec(1) + templateSpec(1) + 256 mezz reads.
	const numFixed = 4
	stageA := make([]common.Hash, 0, numFixed+256)
	stageA = append(stageA,
		CurveSlot(poolHash),
		common.BigToHash(new(big.Int).Add(CurveSlot(poolHash).Big(), big.NewInt(1))),
		PoolSpecsSlot(poolHash),
		TemplateSlot(poolIdx),
	)
	for lobby16 := int16(LobbyKey(FullTickWindow.MinTick)); lobby16 <= int16(LobbyKey(FullTickWindow.MaxTick)); lobby16++ {
		probeTick := int32(int8(lobby16)) << 16
		stageA = append(stageA, MezzSlot(poolHash, probeTick))
	}

	wordsA, err := t.Fetcher.ReadSlotsBatch(ctx, stageA, blockNum)
	if err != nil {
		return nil, fmt.Errorf("stage A batch: %w", err)
	}

	curve := DecodeCurve(slotWord(wordsA[0]), slotWord(wordsA[1]))
	poolSpec := DecodePoolSpec(slotWord(wordsA[2]))
	templateSpec := DecodePoolSpec(slotWord(wordsA[3]))
	mezzWords := wordsA[numFixed:]

	// ---- Stage B: terminus reads for every non-empty mezz word.
	type mezzHit struct {
		lobby   int8
		mezzBit uint8
	}
	var hits []mezzHit
	stageB := make([]common.Hash, 0)
	for i, mezz := range mezzWords {
		if mezz == nil || mezz.Sign() == 0 {
			continue
		}
		lobby := int8(int16(LobbyKey(FullTickWindow.MinTick)) + int16(i))
		for _, mezzBit := range setBitPositions(mezz) {
			mezzKey := WeldLobbyMezz(lobby, mezzBit)
			probeTickTerm := int32(mezzKey) << 8
			stageB = append(stageB, TerminusSlot(poolHash, probeTickTerm))
			hits = append(hits, mezzHit{lobby: lobby, mezzBit: mezzBit})
		}
	}

	wordsB, err := t.Fetcher.ReadSlotsBatch(ctx, stageB, blockNum)
	if err != nil {
		return nil, fmt.Errorf("stage B batch: %w", err)
	}

	var activeTicks []int32
	for i, term := range wordsB {
		if term == nil || term.Sign() == 0 {
			return nil, fmt.Errorf(
				"mezz bit set but terminus empty at lobby=%d mezz=%d",
				hits[i].lobby, hits[i].mezzBit,
			)
		}
		for _, termBit := range setBitPositions(term) {
			tick := WeldLobbyMezzTerm(hits[i].lobby, hits[i].mezzBit, termBit)
			if tick < FullTickWindow.MinTick || tick > FullTickWindow.MaxTick {
				continue
			}
			activeTicks = append(activeTicks, tick)
		}
	}
	sort.Slice(activeTicks, func(i, j int) bool { return activeTicks[i] < activeTicks[j] })

	// ---- Stage C: per active tick, read level + 4 knockout slots.
	stageC := make([]common.Hash, 0, 5*len(activeTicks))
	for _, tick := range activeTicks {
		stageC = append(stageC,
			LevelSlot(poolHash, tick),
			KnockoutPivotSlot(poolHash, true, tick),
			KnockoutMerkleSlot(poolHash, true, tick),
			KnockoutPivotSlot(poolHash, false, tick),
			KnockoutMerkleSlot(poolHash, false, tick),
		)
	}

	wordsC, err := t.Fetcher.ReadSlotsBatch(ctx, stageC, blockNum)
	if err != nil {
		return nil, fmt.Errorf("stage C batch: %w", err)
	}

	levels := make([]TrackedLevel, 0, len(activeTicks))
	knockouts := make([]TrackedKnockout, 0, len(activeTicks))
	for i, tick := range activeTicks {
		base := i * 5
		levels = append(levels, TrackedLevel{
			Tick:  tick,
			Level: DecodeBookLevel(slotWord(wordsC[base])),
		})
		k := TrackedKnockout{
			Tick:      tick,
			BidPivot:  DecodeKnockoutPivot(slotWord(wordsC[base+1])),
			BidMerkle: DecodeKnockoutMerkle(slotWord(wordsC[base+2])),
			AskPivot:  DecodeKnockoutPivot(slotWord(wordsC[base+3])),
			AskMerkle: DecodeKnockoutMerkle(slotWord(wordsC[base+4])),
		}
		if hasTrackedKnockout(k) {
			knockouts = append(knockouts, k)
		}
	}

	return &TrackerExtra{
		Base:           orderedBase,
		Quote:          orderedQuote,
		PoolIdx:        poolIdx,
		PoolHash:       poolHash,
		Curve:          curve,
		PoolSpec:       poolSpec,
		TemplateSpec:   templateSpec,
		PoolParams:     poolSpec.ToPoolParams(),
		TemplateParams: templateSpec.ToPoolParams(),
		ActiveTicks:    activeTicks,
		Levels:         levels,
		Knockouts:      knockouts,
	}, nil
}

// Refresh reuses prev when the curve fingerprint (slot0+slot1) is unchanged,
// avoiding the ~270-call full reload. Returns (extra, changed, err): extra is
// always non-nil on success; changed=false means prev was reused as-is.
//
// Caveat: liquidity mints/burns inside an already-active mezz word do not move
// the curve, so this is sufficient for swap-routing freshness but not for
// bit-exact tick distribution. Pair with a periodic full reload if needed.
func (t *StateTracker) Refresh(
	ctx context.Context,
	prev *TrackerExtra,
	blockNum *big.Int,
) (*TrackerExtra, bool, error) {
	if prev == nil {
		return nil, false, fmt.Errorf("prev is nil")
	}

	curve, err := t.readCurve(ctx, prev.PoolHash, blockNum)
	if err != nil {
		return nil, false, fmt.Errorf("read curve: %w", err)
	}

	if curveEqual(curve, prev.Curve) {
		return prev, false, nil
	}

	extra, err := t.Load(ctx, prev.Base, prev.Quote, prev.PoolIdx, blockNum)
	if err != nil {
		return nil, false, err
	}
	return extra, true, nil
}

func curveEqual(a, b CurveState) bool {
	return a.SeedDeflator == b.SeedDeflator &&
		a.ConcGrowth == b.ConcGrowth &&
		bigEqual(a.PriceRoot, b.PriceRoot) &&
		bigEqual(a.AmbientSeeds, b.AmbientSeeds) &&
		bigEqual(a.ConcLiq, b.ConcLiq)
}

func bigEqual(a, b *big.Int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Cmp(b) == 0
}

func normalizePair(base, quote common.Address) (common.Address, common.Address) {
	if bytes.Compare(base[:], quote[:]) > 0 {
		return quote, base
	}

	return base, quote
}

func (t *StateTracker) readCurve(ctx context.Context, poolHash common.Hash, blockNum *big.Int) (CurveState, error) {
	slot0, err := t.Fetcher.readSlot(ctx, CurveSlot(poolHash), blockNum)
	if err != nil {
		return CurveState{}, err
	}

	slot1Key := common.BigToHash(new(big.Int).Add(CurveSlot(poolHash).Big(), big.NewInt(1)))
	slot1, err := t.Fetcher.readSlot(ctx, slot1Key, blockNum)
	if err != nil {
		return CurveState{}, err
	}

	return DecodeCurve(slotWord(slot0), slotWord(slot1)), nil
}

func (t *StateTracker) readPoolSpec(ctx context.Context, poolHash common.Hash, blockNum *big.Int) (PoolSpec, error) {
	word, err := t.Fetcher.readSlot(ctx, PoolSpecsSlot(poolHash), blockNum)
	if err != nil {
		return PoolSpec{}, err
	}

	return DecodePoolSpec(slotWord(word)), nil
}

func (t *StateTracker) readTemplateSpec(ctx context.Context, poolIdx uint64, blockNum *big.Int) (PoolSpec, error) {
	word, err := t.Fetcher.readSlot(ctx, TemplateSlot(poolIdx), blockNum)
	if err != nil {
		return PoolSpec{}, err
	}

	return DecodePoolSpec(slotWord(word)), nil
}

func (t *StateTracker) readLevel(
	ctx context.Context,
	poolHash common.Hash,
	tick int32,
	blockNum *big.Int,
) (BookLevel, error) {
	word, err := t.Fetcher.readSlot(ctx, LevelSlot(poolHash, tick), blockNum)
	if err != nil {
		return BookLevel{}, err
	}

	return DecodeBookLevel(slotWord(word)), nil
}

func (t *StateTracker) readKnockout(
	ctx context.Context,
	poolHash common.Hash,
	tick int32,
	blockNum *big.Int,
) (TrackedKnockout, error) {
	bidPivotWord, err := t.Fetcher.readSlot(ctx, KnockoutPivotSlot(poolHash, true, tick), blockNum)
	if err != nil {
		return TrackedKnockout{}, err
	}
	bidMerkleWord, err := t.Fetcher.readSlot(ctx, KnockoutMerkleSlot(poolHash, true, tick), blockNum)
	if err != nil {
		return TrackedKnockout{}, err
	}
	askPivotWord, err := t.Fetcher.readSlot(ctx, KnockoutPivotSlot(poolHash, false, tick), blockNum)
	if err != nil {
		return TrackedKnockout{}, err
	}
	askMerkleWord, err := t.Fetcher.readSlot(ctx, KnockoutMerkleSlot(poolHash, false, tick), blockNum)
	if err != nil {
		return TrackedKnockout{}, err
	}

	return TrackedKnockout{
		Tick:      tick,
		BidPivot:  DecodeKnockoutPivot(slotWord(bidPivotWord)),
		BidMerkle: DecodeKnockoutMerkle(slotWord(bidMerkleWord)),
		AskPivot:  DecodeKnockoutPivot(slotWord(askPivotWord)),
		AskMerkle: DecodeKnockoutMerkle(slotWord(askMerkleWord)),
	}, nil
}

func hasTrackedKnockout(k TrackedKnockout) bool {
	return k.BidPivot.Lots.Sign() > 0 ||
		k.BidMerkle.MerkleRoot.Sign() > 0 ||
		k.AskPivot.Lots.Sign() > 0 ||
		k.AskMerkle.MerkleRoot.Sign() > 0
}

func slotWord(word *big.Int) [32]byte {
	var out [32]byte
	if word != nil {
		word.FillBytes(out[:])
	}

	return out
}
