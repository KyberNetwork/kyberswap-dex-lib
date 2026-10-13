package tickermemev4

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType        = valueobject.ExchangeUniswapV4TickerMeme
	basisPoints    = 10_000
	maxTradeFeeBps = 2_000
	// Conservative allowance for afterSwap's take and three fee ledgers. Calibrate
	// against production swaps once the launch factory is wired on Monad.
	afterSwapGas = 100_000
)

var (
	HookAddress              = common.HexToAddress("0x223F8B1b43AEEc86d2659cCe36169f72f7FC2044")
	ImplementationAddress    = common.HexToAddress("0x2D96E4e89D37551Ff55eB9cfEaD78DA1278D5986")
	ImplementationCodeHash   = common.HexToHash("0x94a20f9305eeae246d3ec31488868fc17f81caf4a7ad2b05f528d04c55c4c382")
	implementationSlot       = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	proxyCodeHash            = common.HexToHash("0x824a513357c89e28b2b23a5d727a23bdb3b69db732883d999165af4e064e3120")
	ErrUnsupportedDeployment = errors.New("ticker-meme-v4: unsupported deployment or implementation")
	ErrUntracked             = errors.New("ticker-meme-v4: missing verified hook state")
	ErrInvalidTerms          = errors.New("ticker-meme-v4: invalid launch fees")
)
