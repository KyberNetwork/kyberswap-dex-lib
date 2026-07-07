package brownfiv3

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "brownfi-v3"

	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodPriceFeedIds   = "priceFeedIds"
	factoryMethodPriceOracle    = "priceOracle"
	factoryMethodPairConfig     = "pairConfig"
	factoryMethodGetAmmPrice    = "getAmmPrice"
	factoryMethodGetSwapPrices  = "getSwapPrices"
	factoryMethodIsPaused       = "isPaused"

	pairMethodToken0          = "token0"
	pairMethodToken1          = "token1"
	pairMethodGetReserves     = "getReserves"
	pairMethodQuoteTokenIndex = "quoteTokenIndex"

	pairConfigMethodGetConfig = "getConfig"

	oracleMethodGetUpdateFee = "getUpdateFee"

	pythDefaultUrl = "https://hermes.pyth.network/v2/updates/price/latest"

	ttlStatic      = time.Hour
	maxAge         = 15 * time.Second
	parsedDecimals = 18

	defaultGas = 533708
)

var (
	Router = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDArbitrumOne:     common.HexToAddress("0x96cE2973581C5bF362e0fc9f40e6B5f12AA59b61"),
		valueobject.ChainIDAvalancheCChain: common.HexToAddress("0x123AE7196548ED7370854F91f153cd4e5918A011"),
		valueobject.ChainIDBSC:             common.HexToAddress("0x90800Da4dEa18bE4B1195F8A9e348870F2C6B8FF"),
		valueobject.ChainIDBase:            common.HexToAddress("0x38c91c64169c7B5eBe02DcE39060B6180065C38d"),
		valueobject.ChainIDBerachain:       common.HexToAddress("0x63D8C045ebEc54c4C4bb3e24cA3bf7FD4fFd209a"),
		valueobject.ChainIDEthereum:        common.HexToAddress("0x92927Ff9420aF3347Ae25ad618Eb844E78EFe8E1"),
		valueobject.ChainIDHyperEVM:        common.HexToAddress("0xc0E55d0085266E9A33456610E08172f9c173F908"),
		valueobject.ChainIDLinea:           common.HexToAddress("0xB3c31fDc0a22D5725C47B1fC430F5B87353D8C3e"),
		valueobject.ChainIDMonad:           common.HexToAddress("0x43C08a1689e81EFF83bbAfA35617CcCf2EF463fD"),
		valueobject.ChainIDPolygon:         common.HexToAddress("0x6739e1b16AC12cae7A233d9804DB8128DeA9886A"),
	}

	q64        = big256.U2Pow64
	q64x2      = new(uint256.Int).Mul(q64, big256.U2)
	q128       = big256.U2Pow128
	precisionU = big256.TenPow(8) // 10^8, PRECISION = 1e8

	ErrResponseRaced         = errors.New("response raced")
	ErrFailToFetchPriceFeeds = errors.New("fail to fetch price feeds")

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidPrices            = errors.WithMessage(pool.ErrUnsupported, "invalid prices")
	ErrZeroPythPrice            = errors.New("zero pyth price")
	ErrZeroAdjPrice             = errors.New("zero adj price")
	ErrSpreadExceedsThreshold   = errors.New("spread exceeds dis threshold")
	ErrBuySpreadTooLarge        = errors.New("buy spread >= precision")
	ErrZeroPreTradePrice        = errors.New("zero pre-trade price")
	ErrZeroOutputPrice          = errors.New("zero output price")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrMathUnderflow            = errors.New("MATH_UNDERFLOW")
	ErrZeroDenominator          = errors.New("ZERO_DENOMINATOR")
	ErrZeroOutputAmount         = errors.New("ZERO_OUTPUT_AMOUNT")
	ErrPoolPastGamma            = errors.New("POOL_PAST_GAMMA")
	ErrCutoffLimitReached       = errors.New("CUTOFF_LIMIT_REACHED")
	ErrCutoffInputLimitReached  = errors.New("CUTOFF_INPUT_LIMIT_REACHED")
	ErrInvalidPrice             = errors.New("INVALID_PRICE")
)
