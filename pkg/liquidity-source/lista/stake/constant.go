package stake

import "errors"

const (
	DexType = "lista-stake"

	// gasDeposit: measured 55,237 gas for the executeListaStakeManager delegatecall (approve +
	// deposit() + balance diff + emit) on a successful mainnet-fork simulation
	// (sim f306ac58-3b0a-48f8-a3b1-b55a45562884); padded for a colder first-time-depositor SSTORE.
	gasDeposit         = 80000
	gasInstantWithdraw = 250000

	// defaultDepositReserve is a large sentinel for the deposit-direction (slisBNB out) reserve:
	// deposit() always mints, so it isn't liquidity-capped like the withdraw buffer is.
	defaultDepositReserve = "100000000000000000000000000"
)

var (
	ErrPoolPaused                 = errors.New("pool is paused")
	ErrInstantWithdrawNotEligible = errors.New("instant withdraw not eligible: caller not whitelisted")
	ErrAmountTooSmall             = errors.New("amount too small")
	ErrInsufficientLiquidity      = errors.New("insufficient liquidity in withdraw buffer")
	ErrInvalidToken               = errors.New("invalid token")
	ErrZeroAmountOut              = errors.New("zero amount out")
	ErrOverflow                   = errors.New("overflow")
)
