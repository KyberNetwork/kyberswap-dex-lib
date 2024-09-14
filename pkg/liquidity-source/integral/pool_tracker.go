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
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (u *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("%s: Start getting new state of pool (address: %s)", u.config.DexID, p.Address)

	var (
		reserves [2]*big.Int

		priceInfo    PriceInfo
		averagePrice = big.NewInt(0)
		spotPrice    = big.NewInt(0)

		swapFee *big.Int

		xDecimals uint8
		yDecimals uint8

		oracle common.Address
	)

	rpcRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{ABI: reserveABI, Target: p.Address, Method: libraryGetReservesMethod}, []interface{}{&reserves})
	rpcRequest.AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairSwapFeeMethod}, []interface{}{&swapFee})
	rpcRequest.AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairOracleMethod}, []interface{}{&oracle})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch basic pool data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	rpcRequest = u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleGetPriceInfoMethod}, []interface{}{&priceInfo})
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleGetSpotPriceMethod}, []interface{}{&spotPrice})
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleXDecimalsMethod}, []interface{}{&xDecimals})
	rpcRequest.AddCall(&ethrpc.Call{ABI: oracleABI, Target: oracle.Hex(), Method: oracleYDecimalsMethod}, []interface{}{&yDecimals})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch oracle data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	rpcRequest = u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(
		&ethrpc.Call{
			ABI:    oracleABI,
			Target: oracle.Hex(),
			Method: oracleGetAveragePriceMethod,
			Params: []interface{}{priceInfo.PriceAccumulator, priceInfo.PriceTimestamp},
		}, []interface{}{&averagePrice})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch average price (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	extraData := IntegralPair{
		SwapFee:      ToUint256(swapFee),
		AveragePrice: ToUint256(averagePrice),
		SpotPrice:    ToUint256(spotPrice),
		X_Decimals:   uint64(xDecimals),
		Y_Decimals:   uint64(yDecimals),
	}
	extraBytes, err := json.Marshal(extraData)
	if err != nil {
		logger.Errorf("%s: failed to marshal extra data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	if len(p.Tokens) == 2 {
		p.Tokens[0].Decimals = xDecimals
		p.Tokens[1].Decimals = yDecimals
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves([]string{reserves[0].String(), reserves[1].String()})
	p.SwapFee = float64(swapFee.Uint64()) / precison.Float64()

	logger.Infof("%s: Pool state updated successfully (address: %s)", u.config.DexID, p.Address)

	return p, nil
}
