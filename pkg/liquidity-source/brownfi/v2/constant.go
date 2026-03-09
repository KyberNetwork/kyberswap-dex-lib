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
		valueobject.ChainIDArbitrumOne: common.HexToAddress("0x3240853b71c89209ea8764CDDfA3b81766553E55"),
		valueobject.ChainIDBase:        common.HexToAddress("0x454d337F8afb2dF8168547ab85d937D2445Df47a"),
		valueobject.ChainIDBerachain:   common.HexToAddress("0xa8e42eB1C8aC6228BE39522728E18e7F6d69443c"),
		valueobject.ChainIDBSC:         common.HexToAddress("0xD3F729D909a7E84669A35c3F25b37b4AC3487784"),
		valueobject.ChainIDHyperEVM:    common.HexToAddress("0x0A461D280891167Ee8391f4F0c03EECaa39ae632"),
		valueobject.ChainIDLinea:       common.HexToAddress("0x3F0bBeEdEa5E5F63a14cBdA82718d4f25501fBeA"),
		valueobject.ChainIDMonad:       common.HexToAddress("0xD3F729D909a7E84669A35c3F25b37b4AC3487784"),
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
