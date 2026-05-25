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

	defaultGas = 443940
)

var (
	Router = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDBerachain: common.HexToAddress("0xFB473aEAe9b0d03c6974BCf5f2B67dA4AF7F6043"),
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
)
