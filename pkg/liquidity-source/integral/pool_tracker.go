package integral

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
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
		reserves = [2]*big.Int{ZERO, ZERO}

		poolState = [6]*big.Int{ZERO, ZERO, ZERO, ZERO, ZERO, ZERO}
		// uint256 price,
		// uint256 fee,
		// uint256 limitMin0,
		// uint256 limitMax0,
		// uint256 limitMin1,
		// uint256 limitMax1

		token0 = common.HexToAddress(p.Tokens[0].Address)
		token1 = common.HexToAddress(p.Tokens[1].Address)

		isPairEnabled bool

		// uint256 xDecimals,
		// uint256 yDecimals,
		// uint256 price
		pairInfo         = [3]interface{}{0, 0}
		invertedPairInfo = [3]interface{}{0, 0}
	)

	rpcRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPoolStateMethod,
		Params: []interface{}{token0, token1},
	}, []interface{}{&poolState})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerIsPairEnabledMethod,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []interface{}{&isPairEnabled})

	rpcRequest.AddCall(&ethrpc.Call{ABI: reserveABI, Target: p.Address, Method: libraryGetReservesMethod}, []interface{}{&reserves})

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch basic pool data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPairByAddressMethod,
		Params: []interface{}{common.HexToAddress(p.Address), false}, // get Price when swap X -> Y
	}, []interface{}{&pairInfo})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPairByAddressMethod,
		Params: []interface{}{common.HexToAddress(p.Address), true}, // get Price when swap Y -> X
	}, []interface{}{&invertedPairInfo})
	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch decimals data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	xDecimals := pairInfo[0].(uint8)
	yDecimals := pairInfo[1].(uint8)

	price := pairInfo[2].(*big.Int)
	invertedPrice := invertedPairInfo[2].(*big.Int)

	var extraData IntegralPair
	json.Unmarshal([]byte(p.Extra), &extraData)

	extraData.Price = number.SetFromBig(price)
	extraData.InvertedPrice = number.SetFromBig(invertedPrice)
	extraData.SwapFee = number.SetFromBig(poolState[1])
	extraData.Token0LimitMin = number.SetFromBig(poolState[2])
	extraData.Token0LimitMax = number.SetFromBig(poolState[3])
	extraData.Token1LimitMin = number.SetFromBig(poolState[4])
	extraData.Token1LimitMax = number.SetFromBig(poolState[5])
	extraData.X_Decimals = uint64(xDecimals)
	extraData.Y_Decimals = uint64(yDecimals)
	extraData.IsEnabled = isPairEnabled

	extraBytes, err := json.Marshal(extraData)
	if err != nil {
		logger.Errorf("%s: failed to marshal extra data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves([]string{reserves[0].String(), reserves[1].String()})
	p.SwapFee = float64(poolState[1].Uint64()) / precision.Float64()

	logger.Infof("%s: Pool state updated successfully (address: %s)", u.config.DexID, p.Address)

	return p, nil
}
