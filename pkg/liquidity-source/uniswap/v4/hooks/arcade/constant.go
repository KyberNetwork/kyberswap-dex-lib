package arcade

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// The ArcadeHook deployments on Arc mainnet. Each one is a single instance shared by
// every launch made on it (per-pool state lives in the hook's mappings, keyed by PoolId
// or launch token). Launches never move between them: a pool belongs to the hook in its
// PoolKey for life.
var (
	// HookV1: permission bitmap 0x3ECE, deployed at block 20030281.
	HookV1 = common.HexToAddress("0x695cfF9C7F11fa87ca05c7b0A0fa64C3554B3eCe")
	// HookV2: permission bitmap 0x3EC2, i.e. v1 without BEFORE_SWAP_RETURNS_DELTA and
	// AFTER_SWAP_RETURNS_DELTA. Deployed at block 21568825.
	HookV2 = common.HexToAddress("0x7706d261f0C370e8f0E273164A603E885C89beC2")

	HookAddresses = []common.Address{HookV1, HookV2}
)

// Generation is what differs between ArcadeHook deployments as far as a swap goes. It is
// decided from the hook address alone, never from RPC. The zero value is v1, so a pool
// whose flag went missing is quoted with the v1 hook fee: too low, never too high.
type Generation struct {
	// NoSwapDelta: the hook never returns a swap delta (its address carries neither
	// RETURNS_DELTA bit). Every graduated pool, PUMP included, charges its trading fee as
	// the pool's own static LP fee, already in the PoolKey: 10_000 pips for PUMP. There
	// is no feeObs oracle to read on such a hook (the getter does not exist).
	NoSwapDelta bool
}

var Generations = map[common.Address]Generation{
	HookV1: {},
	HookV2: {NoSwapDelta: true},
}

// Mirrors of ArcadeHook.sol constants.
const (
	bps = 10_000

	// LaunchMode (CurveState.mode). CLANKER_V3 (2) launches live in V3 pools, never here.
	ModePump    = 0
	ModeClanker = 1
	ModeRwa     = 3

	// Status (CurveState.status).
	StatusCurving           = 0
	StatusGraduationStarted = 1
	StatusGraduated         = 2

	// v1 only. PUMP post-graduation dynamic fee, linear in log-mcap from Max at the
	// graduation mcap tick to Min PumpFeeFloorTicks higher.
	PumpFeeMaxBps     = 100
	PumpFeeMinBps     = 30
	PumpFeeFloorTicks = 23_026

	// v1 only. MAX_TOTAL_TAKE_BPS: hard cap on fee + anti-snipe skim per swap.
	MaxTotalTakeBps = 6_000

	// Per-transaction buy cap on CLANKER / RWA pools: 1% of supply in the first
	// minute after launch, +1% per minute, uncapped after 5 minutes.
	buyCapWindowSeconds = 300
	buyCapStepSeconds   = 60
)

// TotalSupply mirrors ArcadeV4Curve.TOTAL_SUPPLY (1B tokens, 18 decimals).
var TotalSupply = new(big.Int).Mul(big.NewInt(1_000_000_000), big.NewInt(1e18))
