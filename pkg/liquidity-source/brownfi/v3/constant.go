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
		valueobject.ChainIDArbitrumOne:     common.HexToAddress("0x6C0a11200c022AF316F591158B5686931ef93DCf"),
		valueobject.ChainIDAvalancheCChain: common.HexToAddress("0x123AE7196548ED7370854F91f153cd4e5918A011"),
		valueobject.ChainIDBSC:             common.HexToAddress("0x196345d5Bd0415A46331ceb3B9F2EA84061A21fD"),
		valueobject.ChainIDBase:            common.HexToAddress("0xa4f4867c718cbAac6dff34e05b5C35AF5C4E34BA"),
		valueobject.ChainIDBerachain:       common.HexToAddress("0x884F608C5F56B1630AC32E07Ce2c4B75a2360575"),
		valueobject.ChainIDEthereum:        common.HexToAddress("0xd3C1a32FE079BB33c3BDCd8ee4cbfE63e990Bb1D"),
		valueobject.ChainIDHyperEVM:        common.HexToAddress("0x98F6369ecf2A2f7A519773AC40C561701a89828b"),
		valueobject.ChainIDLinea:           common.HexToAddress("0x0bf5Fa65dAE6Db32250E5DC74489cedadd38D338"),
		valueobject.ChainIDMonad:           common.HexToAddress("0x67c71042784a30D828aB684e46Cc79D2040eaF1A"),
		valueobject.ChainIDPolygon:         common.HexToAddress("0xDcE93Ef865D43A6ffFD0309c92070B3859C7f9e5"),
		valueobject.ChainIDRobinhood:       common.HexToAddress("0x2A08Da7B6590ce5D217161F234069CfC54DBe554"),
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
