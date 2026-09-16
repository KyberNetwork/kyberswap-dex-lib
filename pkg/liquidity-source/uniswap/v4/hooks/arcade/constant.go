package arcade

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is the single ArcadeHook instance shared by every Arcade launchpad
// pool on Arc mainnet (per-pool state lives in the hook's mappings, keyed by PoolId
// or launch token). Permission bitmap 0x3ECE.
var HookAddresses = []common.Address{
	common.HexToAddress("0x695cfF9C7F11fa87ca05c7b0A0fa64C3554B3eCe"), // arc, deployed at block 20030281
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

	// PUMP post-graduation dynamic fee, linear in log-mcap from Max at the
	// graduation mcap tick to Min PumpFeeFloorTicks higher.
	PumpFeeMaxBps     = 100
	PumpFeeMinBps     = 30
	PumpFeeFloorTicks = 23_026

	// MAX_TOTAL_TAKE_BPS: hard cap on fee + anti-snipe skim per swap.
	MaxTotalTakeBps = 6_000

	// Per-transaction buy cap on CLANKER / RWA pools: 1% of supply in the first
	// minute after launch, +1% per minute, uncapped after 5 minutes.
	buyCapWindowSeconds = 300
	buyCapStepSeconds   = 60
)

// TotalSupply mirrors ArcadeV4Curve.TOTAL_SUPPLY (1B tokens, 18 decimals).
var TotalSupply = new(big.Int).Mul(big.NewInt(1_000_000_000), big.NewInt(1e18))
