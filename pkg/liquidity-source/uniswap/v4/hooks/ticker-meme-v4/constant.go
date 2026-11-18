package tickermemev4

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const DexType = valueobject.ExchangeUniswapV4TickerMeme

var (
	HookAddress              = common.HexToAddress("0x7cebb02f682cd0403467be1D6dC54EeD66AC2044")
	hookCodeHash             = common.HexToHash("0xa5fda91c938700a6a03b3176c79b9705d0177e889f68970b723fa6c870cc5ab0")
	ErrUnsupportedDeployment = errors.New("ticker-meme-v4: unsupported deployment")
)
