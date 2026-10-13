package tickermemev4

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4TickerMeme

var (
	HookAddress              = common.HexToAddress("0x223F8B1b43AEEc86d2659cCe36169f72f7FC2044")
	ImplementationAddress    = common.HexToAddress("0x2D96E4e89D37551Ff55eB9cfEaD78DA1278D5986")
	implementationSlot       = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	ErrUnsupportedDeployment = errors.New("ticker-meme-v4: unsupported deployment or implementation")
)
