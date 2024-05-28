package syncswapv2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
)

// const (
// 	feeManagerV2AddressDefault = "0x63ad090242b4399691d3c1e2e9df4c2d88906ebb"
// )

type PoolTracker struct {
	config       *syncswap.Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *syncswap.Config,
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
	switch p.Type {
	case PoolTypeSyncSwapV2Classic:
		return d.getClassicPoolState(ctx, p)
	case PoolTypeSyncSwapV2Stable:
		return d.getStablePoolState(ctx, p)
	case PoolTypeSyncSwapV2Aqua:
		return d.getAquaPoolState(ctx, p)
	default:
		err := fmt.Errorf("can not get new pool state of address %s with type %s", p.Address, p.Type)
		logger.Errorf(err.Error())
		return entity.Pool{}, err
	}
}

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

func (d *PoolTracker) getAquaPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
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

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    masterABI,
		Target: d.config.MasterAddress,
		Method: poolMethodGetFeeManager,
		Params: nil,
	}, []interface{}{&feeManagerV2Address})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
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
		ABI:    stablePoolABI,
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
			A:                         params.A,
			D:                         D,
			Gamma:                     params.Gamma,
			SwapFee0To1Min:            swapFee0To1Aqua.MinFee,
			SwapFee0To1Max:            swapFee0To1Aqua.MaxFee,
			SwapFee0To1Gamma:          big.NewInt(int64(swapFee0To1Aqua.Gamma)),
			SwapFee1To0Min:            swapFee1To0Aqua.MinFee,
			SwapFee1To0Max:            swapFee1To0Aqua.MaxFee,
			SwapFee1To0Gamma:          big.NewInt(int64(swapFee1To0Aqua.Gamma)),
			FutureTime:                params.FutureTime.Int64(),
			PriceScale:                priceScale,
			LastPrices:                lastPrices,
			PriceOracle:               priceOracle,
			LpSupply:                  lpSupply,
			XcpProfit:                 xcpProfit,
			VirtualPrice:              virtualPrice,
			AllowedExtraProfit:        big.NewInt(int64(rebalaceParams.AllowedExtraProfit)),
			AdjustmentStep:            big.NewInt(int64(rebalaceParams.AdjustmentStep)),
			MaHalfTime:                big.NewInt(int64(rebalaceParams.MaTime)),
			LastPricesTimestamp:       lastPriceTimestamp.Int64(),
			Token0PrecisionMultiplier: token0PrecisionMultiplier,
			Token1PrecisionMultiplier: token1PrecisionMultiplier,
			VaultAddress:              vaultAddress.Hex(),
			InitialA:                  int64(poolParams.InitialA),
			FutureA:                   int64(poolParams.FutureA),
			InitialGamma:              int64(poolParams.InitialGamma),
			FutureGamma:               int64(poolParams.FutureGamma),
			InitialTime:               int64(poolParams.InitialTime),
			FeeManagerAddress:         feeManagerV2Address.Hex(),
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

func (d *PoolTracker) getStablePoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0                             *big.Int
		token0PrecisionMultiplier, token1PrecisionMultiplier *big.Int
		vaultAddress                                         common.Address
		reserves                                             = make([]*big.Int, len(p.Tokens))
		A                                                    uint64
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[0].Address),
			common.HexToAddress(p.Tokens[1].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee0To1})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[1].Address),
			common.HexToAddress(p.Tokens[0].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee1To0})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken0PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token0PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken1PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token1PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetA,
		Params: nil,
	}, []interface{}{&A})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraStablePool{
		ExtraStablePool: syncswap.ExtraStablePool{
			SwapFee0To1:               swapFee0To1,
			SwapFee1To0:               swapFee1To0,
			Token0PrecisionMultiplier: token0PrecisionMultiplier,
			Token1PrecisionMultiplier: token1PrecisionMultiplier,
			VaultAddress:              vaultAddress.Hex(),
		},
		A: big.NewInt(int64(A)),
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

func (d *PoolTracker) getClassicPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0 *big.Int
		reserves                 = make([]*big.Int, len(p.Tokens))
		vaultAddress             common.Address
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[0].Address),
			common.HexToAddress(p.Tokens[1].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee0To1})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[1].Address),
			common.HexToAddress(p.Tokens[0].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee1To0})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(syncswap.ExtraClassicPool{
		SwapFee0To1:  swapFee0To1,
		SwapFee1To0:  swapFee1To0,
		VaultAddress: vaultAddress.Hex(),
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
