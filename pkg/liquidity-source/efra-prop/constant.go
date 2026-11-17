package efraprop

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	DexType = "efra-prop"

	// defaultGas is the integer average of two test-dex swap measurements on Base (TTWOc/USDC):
	// sell 92184, buy 70207, both stable across 4 runs; (92184+70207)/2 = 81195.5, floored.
	defaultGas = 81_195

	// maxAge bounds quoting a probed ladder. The venue's quotes expire at most 15 blocks
	// (~30s on Base) after the maker's last update, so a ladder probed longer ago may price
	// a curve that has already been replaced.
	maxAge = 30 * time.Second
)

// defaultDest is the KyberSwap executor. The venue keys its gap discount by msg.sender of
// swapExactIn, which on our swap path is the executor helper that calls the venue.
var defaultDest = common.HexToAddress("0x8f10b468b06c6fd214b65f87778827f7d113f996")

var (
	ErrNoUSDC        = errors.New("efra-prop: venue returned a zero USDC")
	ErrInvalidToken  = errors.New("efra-prop: invalid token address")
	ErrNoBlockNumber = errors.New("efra-prop: multicall returned no block number")
)
