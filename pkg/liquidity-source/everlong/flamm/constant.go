package everlongflamm

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// DexType is the exchange and pool type of every Efficient pool, whatever its hooks, feeds and lending venues.
const DexType = valueobject.ExchangeEverlongFlamm

const (
	// MaxAge bounds how long a probed state stays routable while StaleCheck is on: every pool prices against
	// oracle feeds.
	MaxAge = 30 * time.Second

	// refreshSeconds is how soon after a refresh the tracker asks for the next, with no event: a feed round, an
	// operator's tuning and a venue's state move the quote and log from elsewhere than the pool.
	refreshSeconds int64 = 10

	// maxVenues bounds the Router records read per pool; past it the pool is not routed, as a record unread could
	// be of a kind the chain does not route.
	maxVenues = 64
	// venueWindow is how many records the read after the first asks for beside the count; more take one read more.
	venueWindow = 8

	// The sampler (FlammSampler) prices a pool's directions in one eth_call, each call to the pool or hook under a
	// per-probe cap. defaultProbeGas is the cap a pool starts at; a pool whose probe starves there is asked at twice
	// its cap, up to maxProbeGas. defaultSampleGas is the call's own gas, stated, since a node's default is its own.
	// samplerReserveGas is what the sampler keeps to encode its answer: it calls the pool or hook only with the cap,
	// its 64th and that much left, so a low gas cap gives a shorter ladder, not a failed call. minSampleGas is the
	// least call gas that passes that gate at maxProbeGas, with 100,000 for the call's own entry.
	defaultProbeGas   uint64 = 5_000_000
	maxProbeGas       uint64 = 20_000_000
	defaultSampleGas  uint64 = 45_000_000
	samplerReserveGas uint64 = 300_000
	minSampleGas             = maxProbeGas*64/63 + samplerReserveGas + 100_000
	// samplerFlipShift: the sampler bisects a change of route label while its bracket is wider than its lower end
	// over 2^7. samplerEdgeProbes is how often it bisects towards the first refusal above a price. samplerGapPpm: it
	// splits a span whose chord may lose more than that many parts per million of the output to the pool's curve.
	samplerFlipShift  = 7
	samplerEdgeProbes = 3
	samplerGapPpm     = 100
	// maxProbesPerDirection is a direction's probe budget in one sampler call, maxProbesPerRefresh a pool's, shared
	// out evenly; the probe above a tier-labelled top may add one past either. maxSampledDirections is how many
	// directions share it at the least a walk can use, four each: the rest are not routed that refresh.
	// firstRefreshSeeds is how many sizes a direction with no ladder yet is seeded at.
	maxProbesPerDirection = 32
	maxProbesPerRefresh   = 64
	maxSampledDirections  = 16
	firstRefreshSeeds     = 32
	// samplerTimeout bounds one sampler call. A node that refuses the sampler samplerFailuresToCooldown times in a
	// row, no pool answering in between, leaves every refresh to the plain path for samplerRetryAfter.
	samplerTimeout            = 3 * time.Second
	samplerRetryAfter         = 10 * time.Minute
	samplerFailuresToCooldown = 8

	// The plain path, where a node does not run the sampler: each probe, request read and leverage-state read is its
	// own eth_call in a JSON-RPC batch, plainProbeGas each, as no cap is raised there, and a direction is probed at
	// plainSeedsPerDirection sizes at most, with no bisection.
	plainProbeGas          uint64 = 15_000_000
	plainSeedsPerDirection int    = 16

	// defaultHaircutBps is taken off every amount interpolated between two probed sizes (never off a probed one).
	defaultHaircutBps uint64 = 1

	// defaultRungs is how many sizes a direction's ladder is seeded at past its first, on every refresh after a
	// pool's first, beside one size under its last first point; minRungs and maxRungs bound the config's.
	defaultRungs = 20
	minRungs     = 4
	maxRungs     = 24

	// venueSwap is the adapter's kind for every pair of every pool: EverlongFlammAdapter kind 0, `IFLAMM.swap`.
	venueSwap uint8 = 0

	// maxTokens is the most tokens the trade mask names (bit 16*i + j).
	maxTokens = 16

	// pairHookTokens bounds the tokens a pair hook may price (FLAMMStore.PAIR_HOOK_TOKENS): its mask names pairs of
	// tokens 0 to 13 only.
	pairHookTokens = 14

	// laneVersionShift and laneVersionMask read the cheap lane's version (FLAMMStore.LANE_VERSION, bits 97-103 of
	// `lane()`'s laneConfig): nonzero once a listed front opened the lane, which then prices the swap hook's pairs
	// for the pool's life.
	laneVersionShift = 97
	laneVersionMask  = 0x7f

	// maxPointBits bounds a probed amount, so products of two stay inside 256 bits.
	maxPointBits = 120

	// maxLadderPoints bounds a ladder: Ladder.Flat holds one bit per span.
	maxLadderPoints = 64

	// dustOut is the output under which a refusal ends nothing: the pool's band test rounds a whole-unit output, so a
	// fill of n units can miss the band by 1/n, and small sizes alternate between a price and a refusal.
	dustOut = 10_000

	// aboveShift places the probe above a tier-labelled top, at used + (used >> aboveShift) + 1: settle keeps a
	// tier-labelled point only where one of the points it takes from the probes is priced at least
	// used >> aboveShift above it.
	aboveShift = 10

	// minSpreadWad and capMarginWad are the row gate's floors (rowOk): the least lever-down spread, 1 bp, and the
	// least room each cap leaves inside the band, 0.1 %.
	minSpreadWad = 100_000_000_000_000
	capMarginWad = 1_000_000_000_000_000

	// shapeEverlong is the one output shape a hook family can state (HookFamily.Shape): EverlongHook's, the better of
	// two routes, its curve's and its leverage row's, each fill labelled by the fee rate it reports.
	shapeEverlong = "everlong"

	// paramsWords is how many words a hook's params() answers; words 27-34 are its leverage row.
	paramsWords = 35

	// requestWords is how many words the pool's pairRequest answers, levStateWords how many the hook's levState.
	requestWords  = 26
	levStateWords = 8
)

// levStateSelector is EverlongHook's levState(PairRequest): sent as raw call data, the selector and then the 26
// words pairRequest answered, as an ABI entry taking uint256[26] would carry another selector.
var levStateSelector = []byte{0xd3, 0x8f, 0x9b, 0xd9}

// samplerTag starts every answer of the sampler: bytes4(keccak256("FlammSampler")), so a caller can tell its code
// ran.
var samplerTag = [4]byte{0xed, 0x7b, 0x4a, 0xc9}

// starvedReverts are the reverts a probe ran short of gas with, ReadStarved() of a venue read and GasShort(): a
// starved probe, which says nothing of the pool.
var starvedReverts = [][]byte{{0x67, 0x52, 0x1d, 0xd6}, {0x88, 0x2f, 0x2c, 0xb8}}

// defaultTaker is the KyberSwap executor, the caller of the pool's swap in a route: a hook prices the swap for its
// caller, so every probe is sent from it.
var defaultTaker = common.HexToAddress("0x8f10b468b06c6fd214b65f87778827f7d113f996")

// erc1967Slot is the ERC-1967 implementation slot.
var erc1967Slot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

// defaultCodeVersions are the pool implementations quoted when the config names none. Every implementation
// change reports a higher `poolCodeVersion()`, so an upgrade the config does not name leaves the pools unrouted.
var defaultCodeVersions = []uint32{1}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrInvalidState = errors.New("invalid pool state")
	ErrRead         = errors.New("a pool read failed")
)
