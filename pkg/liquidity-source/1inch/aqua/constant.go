package oneinchaqua

import (
	"errors"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeOneInchAqua

	// tokensCountDocked is the Aqua marker of a closed strategy.
	tokensCountDocked = 0xff

	// Gas of one hop through executeAqua: traced fills on 8 chains, minus ~50k measured for executeAqua
	// against executeGeneric on an Ethereum fork.
	defaultGas             = 150_000
	defaultGasFeeCollected = 170_000
)

var (
	ErrInvalidToken       = errors.New("1inch-aqua: invalid token")
	ErrInvalidAmountIn    = errors.New("1inch-aqua: invalid amount in")
	ErrInvalidAmountOut   = errors.New("1inch-aqua: invalid amount out")
	ErrNoAmountIn         = errors.New("1inch-aqua: no amount in gives the amount out")
	ErrNoConvergence      = errors.New("1inch-aqua: amount in search did not converge")
	ErrZeroAmountOut      = errors.New("1inch-aqua: zero amount out")
	ErrStrategyTerminal   = errors.New("1inch-aqua: strategy is docked or inactive")
	ErrNoWalletSnapshot   = errors.New("1inch-aqua: no maker wallet snapshot yet")
	ErrStateMixedBlock    = errors.New("1inch-aqua: raw balances and wallet snapshot are from different blocks")
	ErrInsufficientRaw    = fmt.Errorf("1inch-aqua: amount out exceeds the strategy balance: %w", pool.ErrNotEnoughInventory)
	ErrInsufficientWallet = fmt.Errorf("1inch-aqua: amount out exceeds the maker wallet limit: %w", pool.ErrNotEnoughInventory)
	ErrRawRead            = errors.New("1inch-aqua: rawBalances read failed")
	ErrInvalidState       = errors.New("1inch-aqua: invalid pool state")
	ErrFeeShortfall       = errors.New("1inch-aqua: maker cannot cover the protocol fee on the launch router")
	ErrNonCanonical       = errors.New("1inch-aqua: strategy bytes are not the canonical abi.encode(Order)")
	ErrBadProgram         = errors.New("1inch-aqua: unsupported or malformed program")
	ErrOverflow           = errors.New("1inch-aqua: arithmetic overflow")
	ErrZeroBalance        = errors.New("1inch-aqua: xycSwap needs both balances above zero")
)
