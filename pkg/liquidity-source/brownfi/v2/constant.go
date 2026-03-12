package brownfiv2

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "brownfi-v2"

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodPriceFeedIds   = "priceFeedIds"
	factoryMethodPriceOracle    = "priceOracle"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
	pairMethodFee         = "fee"
	pairMethodKappa       = "k"
	pairMethodLambda      = "lambda"

	oracleMethodGetPrice       = "getPrice"
	oracleMethodGetPriceUnsafe = "getPriceUnsafe"
	oracleMethodGetUpdateFee   = "getUpdateFee"

	pythDefaultUrl = "https://hermes.pyth.network/v2/updates/price/latest"

	ttlStatic      = time.Hour
	maxAge         = 15 * time.Second
	parsedDecimals = 18

	defaultGas = 387186
)

var (
	Router = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDArbitrumOne: common.HexToAddress("0x68F963E0CEC0360a003AB8c6D61FA3f39497c463"),
		valueobject.ChainIDBase:        common.HexToAddress("0x454d337F8afb2dF8168547ab85d937D2445Df47a"),
		valueobject.ChainIDBerachain:   common.HexToAddress("0xb91458408dc7bb0561da70ffd89903794eAcDDA7"),
		valueobject.ChainIDBSC:         common.HexToAddress("0xbCE4436BB4F0AdAcdAb3c9d3aEE44059cb9c371B"),
		valueobject.ChainIDHyperEVM:    common.HexToAddress("0x7D06f0ad977B3276da37f2Da4b3a7b2c639244aA"),
		valueobject.ChainIDLinea:       common.HexToAddress("0xf435c8a8b8FFB5236eFF9a955dee41564C84Aa62"),
		valueobject.ChainIDMonad:       common.HexToAddress("0xFaed28e0ffb1C07A2Aa7A41Ae03536dfC12B0db7"),
	}

	q64         = big256.U2Pow64
	q64x2       = new(uint256.Int).Mul(q64, big256.U2)
	q128        = big256.U2Pow128
	dummyMaxAge = bignumber.B2Pow31
	precision   = big256.TenPow(8)

	ErrResponseRaced         = errors.New("response raced")
	ErrFailToFetchPriceFeeds = errors.New("fail to fetch price feeds")

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidPrices            = errors.WithMessage(pool.ErrUnsupported, "invalid prices")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrMax80PercentOfReserve    = errors.New("MAX_80_PERCENT_OF_RESERVE")
)
