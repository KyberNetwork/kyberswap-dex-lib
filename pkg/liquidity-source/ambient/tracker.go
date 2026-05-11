package ambient

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type TrackedLevel struct {
	Tick  int32
	Level BookLevel
}

type TrackerExtra struct {
	Base        common.Address
	Quote       common.Address
	PoolIdx     uint64
	PoolHash    common.Hash
	Curve       CurveState
	PoolSpec    PoolSpec
	PoolParams  PoolParams
	ActiveTicks []int32
	Levels      []TrackedLevel
	MinTick     int32 `json:"minTick,omitempty"`
	MaxTick     int32 `json:"maxTick,omitempty"`
}

type StateTracker struct {
	Client  *ethclient.Client
	DexAddr string
	Fetcher *TickFetcher
}

func NewStateTracker(c *ethclient.Client, dexAddr string) *StateTracker {
	return &StateTracker{
		Client:  c,
		DexAddr: dexAddr,
		Fetcher: NewTickFetcher(c, common.HexToAddress(dexAddr)),
	}
}

func (p PoolSpec) ToPoolParams() PoolParams {
	return PoolParams{
		FeeRate:      p.FeeRate,
		ProtocolTake: p.ProtocolTake,
		TickSize:     p.TickSize,
	}
}

func (t *StateTracker) Load(
	ctx context.Context,
	base, quote common.Address,
	poolIdx uint64,
	blockNum *big.Int,
) (*TrackerExtra, error) {
	return t.LoadWindow(ctx, base, quote, poolIdx, blockNum, FullTickWindow)
}

func (t *StateTracker) LoadCentered(
	ctx context.Context,
	base, quote common.Address,
	poolIdx uint64,
	blockNum *big.Int,
	halfRange int32,
) (*TrackerExtra, error) {
	if halfRange <= 0 {
		return t.Load(ctx, base, quote, poolIdx, blockNum)
	}

	orderedBaseHex, orderedQuoteHex := normalizePair(base.Hex(), quote.Hex())
	orderedBase := common.HexToAddress(orderedBaseHex)
	orderedQuote := common.HexToAddress(orderedQuoteHex)
	poolHash := EncodePoolHash(orderedBase, orderedQuote, poolIdx)

	curve, err := t.readCurve(ctx, poolHash, blockNum)
	if err != nil {
		return nil, fmt.Errorf("centered load: read curve: %w", err)
	}
	if curve.PriceRoot == nil || curve.PriceRoot.Sign() == 0 {
		return &TrackerExtra{
			Base: orderedBase, Quote: orderedQuote,
			PoolIdx: poolIdx, PoolHash: poolHash,
			Curve: curve,
		}, nil
	}

	center := GetTickAtSqrtRatio(curve.PriceRoot)
	minTick := center - halfRange
	maxTick := center + halfRange
	if minTick < FullTickWindow.MinTick {
		minTick = FullTickWindow.MinTick
	}
	if maxTick > FullTickWindow.MaxTick {
		maxTick = FullTickWindow.MaxTick
	}

	return t.LoadWindow(ctx, base, quote, poolIdx, blockNum, TickWindow{MinTick: minTick, MaxTick: maxTick})
}

func (t *StateTracker) LoadWindow(
	ctx context.Context,
	base, quote common.Address,
	poolIdx uint64,
	blockNum *big.Int,
	window TickWindow,
) (*TrackerExtra, error) {
	orderedBaseHex, orderedQuoteHex := normalizePair(base.Hex(), quote.Hex())
	orderedBase := common.HexToAddress(orderedBaseHex)
	orderedQuote := common.HexToAddress(orderedQuoteHex)
	poolHash := EncodePoolHash(orderedBase, orderedQuote, poolIdx)

	minLobby := int16(LobbyKey(window.MinTick))
	maxLobby := int16(LobbyKey(window.MaxTick))
	numLobbies := int(maxLobby-minLobby) + 1

	// ---- Stage A: curve(2) + poolSpec(1) + lobby mezz reads.
	const numFixed = 3
	stageA := make([]common.Hash, 0, numFixed+numLobbies)
	stageA = append(stageA,
		CurveSlot(poolHash),
		common.BigToHash(new(big.Int).Add(CurveSlot(poolHash).Big(), bignum.One)),
		PoolSpecsSlot(poolHash),
	)
	for lobby16 := minLobby; lobby16 <= maxLobby; lobby16++ {
		probeTick := int32(int8(lobby16)) << 16
		stageA = append(stageA, MezzSlot(poolHash, probeTick))
	}

	wordsA, err := t.Fetcher.ReadSlotsBatch(ctx, stageA, blockNum)
	if err != nil {
		return nil, fmt.Errorf("stage A batch: %w", err)
	}

	curve := DecodeCurve(slotWord(wordsA[0]), slotWord(wordsA[1]))
	poolSpec := DecodePoolSpec(slotWord(wordsA[2]))
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
		lobby := int8(minLobby + int16(i))
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
			if tick < window.MinTick || tick > window.MaxTick {
				continue
			}
			activeTicks = append(activeTicks, tick)
		}
	}
	sort.Slice(activeTicks, func(i, j int) bool { return activeTicks[i] < activeTicks[j] })

	// ---- Stage C: per active tick, one level slot.
	stageC := make([]common.Hash, 0, len(activeTicks))
	for _, tick := range activeTicks {
		stageC = append(stageC, LevelSlot(poolHash, tick))
	}

	wordsC, err := t.Fetcher.ReadSlotsBatch(ctx, stageC, blockNum)
	if err != nil {
		return nil, fmt.Errorf("stage C batch: %w", err)
	}

	levels := make([]TrackedLevel, len(activeTicks))
	for i, tick := range activeTicks {
		levels[i] = TrackedLevel{
			Tick:  tick,
			Level: DecodeBookLevel(slotWord(wordsC[i])),
		}
	}

	return &TrackerExtra{
		Base:        orderedBase,
		Quote:       orderedQuote,
		PoolIdx:     poolIdx,
		PoolHash:    poolHash,
		Curve:       curve,
		PoolSpec:    poolSpec,
		PoolParams:  poolSpec.ToPoolParams(),
		ActiveTicks: activeTicks,
		Levels:      levels,
		MinTick:     window.MinTick,
		MaxTick:     window.MaxTick,
	}, nil
}

// Refresh reuses prev when curve + poolSpec fingerprints are unchanged,
// else falls back to LoadWindow. Returns (extra, changed, err).
// Caveat: mints/burns inside an already-active mezz word don't move the curve,
// so pair with periodic full reload for bit-exact tick distribution.
func (t *StateTracker) Refresh(
	ctx context.Context,
	prev *TrackerExtra,
	blockNum *big.Int,
	window TickWindow,
) (*TrackerExtra, bool, error) {
	if prev == nil {
		return nil, false, fmt.Errorf("prev is nil")
	}

	curve, err := t.readCurve(ctx, prev.PoolHash, blockNum)
	if err != nil {
		return nil, false, fmt.Errorf("read curve: %w", err)
	}
	poolSpec, err := t.readPoolSpec(ctx, prev.PoolHash, blockNum)
	if err != nil {
		return nil, false, fmt.Errorf("read pool spec: %w", err)
	}

	if curveEqual(curve, prev.Curve) && poolSpecEqual(poolSpec, prev.PoolSpec) &&
		window.MinTick == prev.MinTick && window.MaxTick == prev.MaxTick {
		return prev, false, nil
	}

	extra, err := t.LoadWindow(ctx, prev.Base, prev.Quote, prev.PoolIdx, blockNum, window)
	if err != nil {
		return nil, false, err
	}
	return extra, true, nil
}

func poolSpecEqual(a, b PoolSpec) bool {
	return a == b
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

func normalizePair(base, quote string) (string, string) {
	if base > quote {
		return quote, base
	}
	return base, quote
}

func (t *StateTracker) readCurve(ctx context.Context, poolHash common.Hash, blockNum *big.Int) (CurveState, error) {
	slot0, err := t.Fetcher.readSlot(ctx, CurveSlot(poolHash), blockNum)
	if err != nil {
		return CurveState{}, err
	}

	slot1Key := common.BigToHash(new(big.Int).Add(CurveSlot(poolHash).Big(), bignum.One))
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

func slotWord(word *big.Int) [32]byte {
	var out [32]byte
	if word != nil {
		word.FillBytes(out[:])
	}

	return out
}
