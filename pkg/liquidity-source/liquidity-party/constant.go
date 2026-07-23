package liquidityparty

import "errors"

const (
	// DexType is the pool-service registration key. It is deliberately "liquidity-party"
	// (not "pool-party", which is an unrelated 1inch-oracle single-token pool already in the repo).
	DexType = "liquidity-party"

	// A single LMSR swap's gas grows linearly with the pool's token count, because swap() recomputes
	// the size metric / EMA across every token. Fitting measured swap traces
	//   (n=10, 149997) (n=20, 165035) (n=30, 180107) (n=50, 210011)
	// gives gas ≈ 135000 + 1500·n (least-squares slope 1500.35, intercept 135028; the clean
	// constants below reproduce all four samples within 107 gas, ≤0.06%). See estimateGas.
	swapBaseGas     int64 = 135000 // fixed swap overhead (kernel exp/ln, ERC20 transfers, fee accounting)
	swapGasPerToken int64 = 1500   // marginal cost per token in the pool

	// poolListBatchSize bounds how many pools GetAllPools returns per discovery page.
	poolListBatchSize = 100
)

var (
	// ErrPoolKilled is returned by the simulator when the pool has been permanently killed
	// (swaps/mints disabled forever; burn-only). Mirrors the on-chain "killed" revert.
	ErrPoolKilled = errors.New("liquidity-party: pool killed")

	// ErrInvalidExtra flags a malformed/empty Extra during simulator construction.
	ErrInvalidExtra = errors.New("liquidity-party: invalid extra")

	// ErrInvalidToken is returned when a requested token is not in the pool.
	ErrInvalidToken = errors.New("liquidity-party: invalid token")

	// ErrSameToken is returned when input and output token are the same (on-chain "same token").
	ErrSameToken = errors.New("liquidity-party: same token")

	// ErrInvalidAmount is returned for a non-positive requested amount (on-chain "invalid amount").
	ErrInvalidAmount = errors.New("liquidity-party: invalid amount")

	// ErrTooSmall mirrors the pool's "too small" revert: the input rounds to zero internally or
	// the priced output is below one wei after fees.
	ErrTooSmall = errors.New("liquidity-party: amount too small")

	// ErrTooLarge mirrors the pool's "too large"/"pool drained" reverts: the swap exceeds the
	// kernel's EXP_LIMIT guard or would drain the output asset's internal balance.
	ErrTooLarge = errors.New("liquidity-party: amount too large or exceeds capacity")

	// ErrUninitialized mirrors the kernel's "uninitialized"/"LMSR: size metric zero" reverts.
	ErrUninitialized = errors.New("liquidity-party: size metric zero (uninitialized)")

	// ErrOverflow flags a 256-bit arithmetic overflow (input decode or an intermediate multiply/add).
	// On-chain these paths run in checked Solidity and revert; the simulator rejects the quote instead
	// of silently wrapping.
	ErrOverflow = errors.New("liquidity-party: arithmetic overflow")

	// ErrInvalidEvent flags a malformed PartyStarted log during event-driven discovery.
	ErrInvalidEvent = errors.New("liquidity-party: invalid PartyStarted event")
)
