package kokonutcrypto

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeKokonutCrypto, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("[kokonut] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		a, dExtra, gamma, feeGamma, midFee, outFee                                                            *uint256.Int
		lastPriceTimestamp, lpSupply, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, maHalfTime *uint256.Int
		priceScale, priceOracle, lastPrices, minRemainingPostRebalanceRatio                                   *uint256.Int
		futureAGammaTime, initialAGammaTime, futureA, initialA                                                uint32
		futureGamma, initialGamma                                                                             uint64
		balances                                                                                              = make([]*uint256.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []any{&a})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []any{&dExtra})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodGamma,
		Params: nil,
	}, []any{&gamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []any{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []any{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []any{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []any{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []any{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodFutureGamma,
		Params: nil,
	}, []any{&futureGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []any{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []any{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodInitialGamma,
		Params: nil,
	}, []any{&initialGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodLastPricesTimestamp,
		Params: nil,
	}, []any{&lastPriceTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []any{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []any{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []any{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []any{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMaHalfTime,
		Params: nil,
	}, []any{&maHalfTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodPriceScale,
		Params: nil,
	}, []any{&priceScale})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodPriceOracle,
		Params: nil,
	}, []any{&priceOracle})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodLastPrices,
		Params: nil,
	}, []any{&lastPrices})

	calls.AddCall(&ethrpc.Call{
		ABI:    cryptoSwap2PoolABI,
		Target: p.Address,
		Method: poolMethodMinRemainingPostRebalanceRatio,
		Params: nil,
	}, []any{&minRemainingPostRebalanceRatio})

	lpToken := p.GetLpToken()
	if len(lpToken) > 0 {
		calls.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: lpToken,
			Method: erc20MethodTotalSupply,
			Params: nil,
		}, []any{&lpSupply})
	}

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    cryptoSwap2PoolABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&balances[i]})
	}

	resp, err := calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	var (
		reserves = make(entity.PoolReserves, len(balances))
	)
	for i := range p.Tokens {
		reserves[i] = u256ToString(balances[i])
	}

	var extra = Extra{
		A:                              u256ToString(a),
		D:                              u256ToString(dExtra),
		Gamma:                          u256ToString(gamma),
		FeeGamma:                       u256ToString(feeGamma),
		MidFee:                         u256ToString(midFee),
		OutFee:                         u256ToString(outFee),
		FutureAGammaTime:               int64(futureAGammaTime),
		FutureA:                        strconv.FormatUint(uint64(futureA), 10),
		FutureGamma:                    strconv.FormatUint(futureGamma, 10),
		InitialAGammaTime:              int64(initialAGammaTime),
		InitialA:                       strconv.FormatUint(uint64(initialA), 10),
		InitialGamma:                   strconv.FormatUint(initialGamma, 10),
		PriceScale:                     u256ToString(priceScale),
		LastPrices:                     u256ToString(lastPrices),
		PriceOracle:                    u256ToString(priceOracle),
		LpSupply:                       u256ToString(lpSupply),
		XcpProfit:                      u256ToString(xcpProfit),
		VirtualPrice:                   u256ToString(virtualPrice),
		AllowedExtraProfit:             u256ToString(allowedExtraProfit),
		AdjustmentStep:                 u256ToString(adjustmentStep),
		MaHalfTime:                     u256ToString(maHalfTime),
		LastPricesTimestamp:            u256ToInt64(lastPriceTimestamp),
		MinRemainingPostRebalanceRatio: u256ToString(minRemainingPostRebalanceRatio),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.Infof("[kokonut] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}

func u256ToString(v *uint256.Int) string {
	if v == nil {
		return zeroString
	}

	return v.String()
}

func u256ToInt64(v *uint256.Int) int64 {
	if v == nil {
		return 0
	}

	return v.ToBig().Int64()
}
