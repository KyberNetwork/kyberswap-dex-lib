package twocryptong

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type PoolTracker struct {
	config       shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

func NewPoolTracker(
	config shared.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	lg.Info("Start updating state ...")
	defer func() { lg.Info("Finish updating state.") }()

	var (
		d, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int

		lastTimestamp, xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, lpSupply *big.Int

		balances = make([]*big.Int, len(p.Tokens))

		numDepCoins = len(p.Tokens) - 1 // other coins will have price based on the 1st coin

		// These 3 slices only has length = number of tokens - 1 (check in the contract)
		priceScales  = make([]*big.Int, numDepCoins)
		priceOracles = make([]*big.Int, numDepCoins)
		lastPrices   = make([]*big.Int, numDepCoins)
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodD,
		Params: nil,
	}, []interface{}{&d})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
		Params: nil,
	}, []interface{}{&feeGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodMidFee,
		Params: nil,
	}, []interface{}{&midFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodOutFee,
		Params: nil,
	}, []interface{}{&outFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
		Params: nil,
	}, []interface{}{&futureAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodFutureAGamma,
		Params: nil,
	}, []interface{}{&futureAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
		Params: nil,
	}, []interface{}{&initialAGammaTime})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodInitialAGamma,
		Params: nil,
	}, []interface{}{&initialAGamma})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodLastTimestamp,
		Params: nil,
	}, []interface{}{&lastTimestamp})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
		Params: nil,
	}, []interface{}{&xcpProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
		Params: nil,
	}, []interface{}{&virtualPrice})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
		Params: nil,
	}, []interface{}{&allowedExtraProfit})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
		Params: nil,
	}, []interface{}{&adjustmentStep})

	calls.AddCall(&ethrpc.Call{
		ABI:    curveTwocryptoNGABI,
		Target: p.Address,
		Method: shared.ERC20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    curveTwocryptoNGABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})
	}

	for i := 0; i < numDepCoins; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    curveTwocryptoNGABI,
			Target: p.Address,
			Method: poolMethodPriceScale,
			Params: nil,
		}, []interface{}{&priceScales[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    curveTwocryptoNGABI,
			Target: p.Address,
			Method: poolMethodPriceOracle,
			Params: nil,
		}, []interface{}{&priceOracles[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    curveTwocryptoNGABI,
			Target: p.Address,
			Method: poolMethodLastPrices,
			Params: nil,
		}, []interface{}{&lastPrices[i]})
	}
	if res, err := calls.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	var extra = Extra{
		InitialA:            number.SetFromBig(new(big.Int).Rsh(initialAGamma, 128)),
		InitialGamma:        new(uint256.Int).And(number.SetFromBig(initialAGamma), PriceMask),
		InitialAGammaTime:   initialAGammaTime.Int64(),
		FutureA:             number.SetFromBig(new(big.Int).Rsh(futureAGamma, 128)),
		FutureGamma:         new(uint256.Int).And(number.SetFromBig(futureAGamma), PriceMask),
		FutureAGammaTime:    futureAGammaTime.Int64(),
		D:                   number.SetFromBig(d),
		LastPricesTimestamp: lastTimestamp.Int64(),
		FeeGamma:            number.SetFromBig(feeGamma),
		MidFee:              number.SetFromBig(midFee),
		OutFee:              number.SetFromBig(outFee),
		LpSupply:            number.SetFromBig(lpSupply),
		XcpProfit:           number.SetFromBig(xcpProfit),
		VirtualPrice:        number.SetFromBig(virtualPrice),
		AllowedExtraProfit:  number.SetFromBig(allowedExtraProfit),
		AdjustmentStep:      number.SetFromBig(adjustmentStep),
	}
	extra.PriceScale = make([]uint256.Int, len(priceScales))
	lo.ForEach(priceScales, func(item *big.Int, i int) { extra.PriceScale[i].SetFromBig(item) })
	extra.PriceOracle = make([]uint256.Int, len(priceOracles))
	lo.ForEach(priceOracles, func(item *big.Int, i int) { extra.PriceOracle[i].SetFromBig(item) })
	extra.LastPrices = make([]uint256.Int, len(lastPrices))
	lo.ForEach(lastPrices, func(item *big.Int, i int) { extra.LastPrices[i].SetFromBig(item) })

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(balances)+1)
	for i := range balances {
		reserves = append(reserves, balances[i].String())
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}
