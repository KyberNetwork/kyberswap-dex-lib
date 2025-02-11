package syncswapv2aqua

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// const (
// 	feeManagerV2AddressDefault = "0x63ad090242b4399691d3c1e2e9df4c2d88906ebb"
// )

type Params struct {
	A          *big.Int
	Gamma      *big.Int
	FutureTime *big.Int
}
type SwapFeeAquaData struct {
	Gamma  uint64   `json:"gamma"`
	MinFee *big.Int `json:"minFee"`
	MaxFee *big.Int `json:"maxFee"`
}
type SwapFeeAqua struct {
	SwapFeeAquaData
}

type RebalanceParams struct {
	AllowedExtraProfit uint64
	AdjustmentStep     uint64
	MaTime             uint32
}

type PoolParams struct {
	InitialA     uint32
	FutureA      uint32
	InitialGamma uint64
	FutureGamma  uint64
	InitialTime  uint32
	FutureTime   uint32
}

type PoolTracker struct {
	config       *syncswapv2.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeSyncSwapV2Aqua, NewPoolTracker)

func NewPoolTracker(
	config *syncswapv2.Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		vaultAddress                                            common.Address
		swapFee0To1Aqua, swapFee1To0Aqua                        SwapFeeAqua
		reserves                                                = make([]*big.Int, len(p.Tokens))
		params                                                  Params
		poolParams                                              PoolParams
		token0PrecisionMultiplier, token1PrecisionMultiplier, D *big.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice   *big.Int
		priceScale, priceOracle, lastPrices                     *big.Int
		rebalaceParams                                          RebalanceParams
		feeManagerV2Address                                     common.Address
	)
	var extra ExtraAquaPool
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return entity.Pool{}, err
	}
	feeManagerV2Address = common.HexToAddress(extra.FeeManagerAddress)

	masterAddress := d.config.MasterAddress[0]
	if extra.MasterAddress != "" {
		masterAddress = extra.MasterAddress
	}

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    masterABI,
		Target: masterAddress,
		Method: poolMethodGetFeeManager,
		Params: nil,
	}, []interface{}{&feeManagerV2Address})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodToken0PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token0PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodToken1PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token1PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaPriceScale,
		Params: nil,
	}, []interface{}{&priceScale})
	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaD,
		Params: nil,
	}, []interface{}{&D})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaParams,
		Params: []interface{}{},
	}, []interface{}{&params})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaPoolParams,
		Params: []interface{}{},
	}, []interface{}{&poolParams})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaPriceOracle,
		Params: nil,
	}, []interface{}{&priceOracle})

	// lpToken := p.GetLpToken()
	// if len(lpToken) > 0 {
	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})
	// }

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaGetLastPrices,
		Params: []interface{}{},
	}, []interface{}{&lastPrices})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaLastPricesTimestamp,
		Params: nil,
	}, []interface{}{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaXcpProfit,
		Params: nil,
	}, []interface{}{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaVirtualPrice,
		Params: nil,
	}, []interface{}{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    aquaPoolABI,
		Target: p.Address,
		Method: poolMethodAquaRebalancingParams,
		Params: nil,
	}, []interface{}{&rebalaceParams})

	calls.AddCall(&ethrpc.Call{
		ABI:    feeManagerV2ABI,
		Target: feeManagerV2Address.Hex(),
		Method: poolMethodAquaGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(p.Address),
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[0].Address),
			common.HexToAddress(p.Tokens[1].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee0To1Aqua})

	calls.AddCall(&ethrpc.Call{
		ABI:    feeManagerV2ABI,
		Target: feeManagerV2Address.Hex(),
		Method: poolMethodAquaGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(p.Address),
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[1].Address),
			common.HexToAddress(p.Tokens[0].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee1To0Aqua})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(
		ExtraAquaPool{
			A:                         uint256.MustFromBig(params.A),
			D:                         uint256.MustFromBig(D),
			Gamma:                     uint256.MustFromBig(params.Gamma),
			SwapFee0To1Min:            uint256.MustFromBig(swapFee0To1Aqua.MinFee),
			SwapFee0To1Max:            uint256.MustFromBig(swapFee0To1Aqua.MaxFee),
			SwapFee0To1Gamma:          uint256.NewInt(swapFee0To1Aqua.Gamma),
			SwapFee1To0Min:            uint256.MustFromBig(swapFee1To0Aqua.MinFee),
			SwapFee1To0Max:            uint256.MustFromBig(swapFee1To0Aqua.MaxFee),
			SwapFee1To0Gamma:          uint256.NewInt(swapFee1To0Aqua.Gamma),
			FutureTime:                params.FutureTime.Int64(),
			PriceScale:                uint256.MustFromBig(priceScale),
			LastPrices:                uint256.MustFromBig(lastPrices),
			PriceOracle:               uint256.MustFromBig(priceOracle),
			LpSupply:                  uint256.MustFromBig(lpSupply),
			XcpProfit:                 uint256.MustFromBig(xcpProfit),
			VirtualPrice:              uint256.MustFromBig(virtualPrice),
			AllowedExtraProfit:        uint256.NewInt(rebalaceParams.AllowedExtraProfit),
			AdjustmentStep:            uint256.NewInt(rebalaceParams.AdjustmentStep),
			MaHalfTime:                uint256.NewInt(uint64(rebalaceParams.MaTime)),
			LastPricesTimestamp:       lastPriceTimestamp.Int64(),
			Token0PrecisionMultiplier: uint256.MustFromBig(token0PrecisionMultiplier),
			Token1PrecisionMultiplier: uint256.MustFromBig(token1PrecisionMultiplier),
			VaultAddress:              vaultAddress.Hex(),
			InitialA:                  int64(poolParams.InitialA),
			FutureA:                   int64(poolParams.FutureA),
			InitialGamma:              int64(poolParams.InitialGamma),
			FutureGamma:               int64(poolParams.FutureGamma),
			InitialTime:               int64(poolParams.InitialTime),
			FeeManagerAddress:         feeManagerV2Address.Hex(),
			MasterAddress:             masterAddress,
		})

	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)
	return p, nil
}
