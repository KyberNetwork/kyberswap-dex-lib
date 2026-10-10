package lotflow

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// NamedError is a simulator error that mirrors one of the contracts' custom errors: ErrorName is the
// Solidity error name and ErrorArgs its arguments as the fixtures print them (uint/int in decimal,
// addresses lowercase 0x-hex). A quote that fails with a NamedError is a quote the chain would revert
// with that same error.
type NamedError interface {
	error
	ErrorName() string
	ErrorArgs() []string
}

type namedErr struct {
	name string
	args []string
}

func (e *namedErr) Error() string {
	if len(e.args) == 0 {
		return "lotflow: " + e.name + "()"
	}
	return fmt.Sprintf("lotflow: %s(%s)", e.name, strings.Join(e.args, ", "))
}
func (e *namedErr) ErrorName() string   { return e.name }
func (e *namedErr) ErrorArgs() []string { return e.args }

// ErrReverted marks a read that reverted on chain (a failed multicall sub-call, or an eth_call the
// node answered with "execution reverted"), as opposed to a transport failure.
var ErrReverted = errors.New("lotflow: execution reverted")

// TransportError is a read that got no answer from the chain (HTTP failure, timeout, a node error
// that is not a revert). Track returns it instead of publishing a state.
type TransportError struct{ Err error }

func (e *TransportError) Error() string { return "lotflow: rpc: " + e.Err.Error() }
func (e *TransportError) Unwrap() error { return e.Err }

func isTransport(err error) bool {
	var t *TransportError
	return errors.As(err, &t)
}

// ErrMulDivZero is FullMath.mulDiv on a zero denominator: the chain reverts (no error data).
var ErrMulDivZero = errors.New("lotflow: mulDiv by zero (reverts on chain)")

// ErrStalePin: Track was pinned to a block more than MaxAgeSec older than the chain head.
var ErrStalePin = errors.New("lotflow: pinned block is stale")

// ErrNumeric is a panic in the simulator's math, recovered at the hook's entry points.
var ErrNumeric = errors.New("lotflow: numeric failure")

// Argument-free errors are sentinels (errors.Is works).
var (
	ErrTradeTooSmall = &namedErr{name: "TradeTooSmall"} // NavJitHookV17
	ErrZeroNav       = &namedErr{name: "ZeroNav"}       // NavGuard
	ErrBadFeedAnswer = &namedErr{name: "BadFeedAnswer"} // NavGuard (stock arg dropped: the sentinel matches any stock)
	ErrLiquidityCast = &namedErr{name: "SafeCastOverflow"}
)

// OutsideBandError: NavJitHookV17 `_requireInBand`, the venue price after the user's swap is more than
// bandBps from NAV.
type OutsideBandError struct{ Px, Nav *big.Int }

func (e *OutsideBandError) Error() string {
	return fmt.Sprintf("lotflow: OutsideBand(%s, %s)", e.Px, e.Nav)
}
func (e *OutsideBandError) ErrorName() string   { return "OutsideBand" }
func (e *OutsideBandError) ErrorArgs() []string { return []string{e.Px.String(), e.Nav.String()} }

// NotProfitableError: NavJitHookV17 `_unwind`, the hook's USDG delta is negative.
type NotProfitableError struct{ Delta *big.Int }

func (e *NotProfitableError) Error() string {
	return fmt.Sprintf("lotflow: NotProfitable(%s)", e.Delta)
}
func (e *NotProfitableError) ErrorName() string   { return "NotProfitable" }
func (e *NotProfitableError) ErrorArgs() []string { return []string{e.Delta.String()} }

// SellExceedsFloatError: NavJitHookV17 `_unwind`, a sell needs more LOT than the PoolManager holds.
type SellExceedsFloatError struct{ Float, Needed *big.Int }

func (e *SellExceedsFloatError) Error() string {
	return fmt.Sprintf("lotflow: SellExceedsFloat(%s, %s)", e.Float, e.Needed)
}
func (e *SellExceedsFloatError) ErrorName() string { return "SellExceedsFloat" }
func (e *SellExceedsFloatError) ErrorArgs() []string {
	return []string{e.Float.String(), e.Needed.String()}
}

// NavMismatchError: NavGuard `_requireAgree`, spot NAV and the Chainlink NAV disagree beyond tolerance.
type NavMismatchError struct{ Spot, Chainlink *big.Int }

func (e *NavMismatchError) Error() string {
	return fmt.Sprintf("lotflow: NavMismatch(%s, %s)", e.Spot, e.Chainlink)
}
func (e *NavMismatchError) ErrorName() string { return "NavMismatch" }
func (e *NavMismatchError) ErrorArgs() []string {
	return []string{e.Spot.String(), e.Chainlink.String()}
}

// FeedCoverageTooLowError: NavGuard `_requireAgree`, the fed share of spot NAV is under minCoverageBps.
type FeedCoverageTooLowError struct{ CoveredBps, MinCoverageBps uint64 }

func (e *FeedCoverageTooLowError) Error() string {
	return fmt.Sprintf("lotflow: FeedCoverageTooLow(%d, %d)", e.CoveredBps, e.MinCoverageBps)
}
func (e *FeedCoverageTooLowError) ErrorName() string { return "FeedCoverageTooLow" }
func (e *FeedCoverageTooLowError) ErrorArgs() []string {
	return []string{fmt.Sprint(e.CoveredBps), fmt.Sprint(e.MinCoverageBps)}
}

// NoExecutableSourceError: NavJitLegs `_pick` / `_sellOne`, no candidate of a constituent can fill.
type NoExecutableSourceError struct{ Stock string }

func (e *NoExecutableSourceError) Error() string {
	return fmt.Sprintf("lotflow: NoExecutableSource(%s)", e.Stock)
}
func (e *NoExecutableSourceError) ErrorName() string   { return "NoExecutableSource" }
func (e *NoExecutableSourceError) ErrorArgs() []string { return []string{strings.ToLower(e.Stock)} }
