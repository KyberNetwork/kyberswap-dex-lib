package integral

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	isFirstRun   bool
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		isFirstRun:   true,
	}, nil
}

func (u *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("%s: Start getting new state of pool (address: %s)", u.config.DexID, p.Address)

	var (
		reserves = [2]*big.Int{ZERO, ZERO}

		priceInfo    PriceInfo
		averagePrice = ZERO
		spotPrice    = ZERO

		swapFee = ZERO

		xDecimals uint8
		yDecimals uint8

		oracle common.Address

		token0 = common.HexToAddress(p.Tokens[0].Address)
		token1 = common.HexToAddress(p.Tokens[1].Address)

		isPairEnabled bool

		token0LimitMin = ZERO
		token1LimitMin = ZERO
	)

	//  Gather basic pool information
	rpcRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetTokenLimitMinMethod,
		Params: []interface{}{token0},
	}, []interface{}{&token0LimitMin})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetTokenLimitMinMethod,
		Params: []interface{}{token1},
	}, []interface{}{&token1LimitMin})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerIsPairEnabledMethod,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&isPairEnabled})

	rpcRequest.AddCall(&ethrpc.Call{ABI: reserveABI, Target: p.Address, Method: libraryGetReservesMethod}, []interface{}{&reserves})
	rpcRequest.AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairSwapFeeMethod}, []interface{}{&swapFee})
	rpcRequest.AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairOracleMethod}, []interface{}{&oracle})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch basic pool data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	// Get priceInfo
	rpcRequest = u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleGetPriceInfoMethod}, []interface{}{&priceInfo})
	if _, err := rpcRequest.Call(); err != nil {
		logger.Errorf("%s: failed to fetch price info (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	// Only if decimals are not available
	if u.isFirstRun {
		rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleXDecimalsMethod}, []interface{}{&xDecimals})
		rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleYDecimalsMethod}, []interface{}{&yDecimals})

		if _, err := rpcRequest.TryAggregate(); err != nil {
			logger.Errorf("%s: failed to fetch decimals data (address: %s, error: %v)", u.config.DexID, p.Address, err)
			return entity.Pool{}, err
		}
	}

	// Get spot price and average price
	rpcRequest = u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    oracleABI,
		Target: oracle.Hex(),
		Method: oracleGetAveragePriceMethod,
		Params: []interface{}{priceInfo.PriceAccumulator, priceInfo.PriceTimestamp},
	}, []interface{}{&averagePrice})
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleGetSpotPriceMethod}, []interface{}{&spotPrice})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch price data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	extraData := IntegralPair{
		SwapFee:        ToUint256(swapFee),
		AveragePrice:   ToUint256(averagePrice),
		SpotPrice:      ToUint256(spotPrice),
		Token0LimitMin: ToUint256(token0LimitMin),
		Token1LimitMin: ToUint256(token1LimitMin),
		X_Decimals:     uint64(xDecimals),
		Y_Decimals:     uint64(yDecimals),
		IsEnabled:      isPairEnabled,
	}
	extraBytes, err := json.Marshal(extraData)
	if err != nil {
		logger.Errorf("%s: failed to marshal extra data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves([]string{reserves[0].String(), reserves[1].String()})
	p.SwapFee = float64(swapFee.Uint64()) / precision.Float64()

	u.isFirstRun = false

	logger.Infof("%s: Pool state updated successfully (address: %s)", u.config.DexID, p.Address)

	return p, nil
}
