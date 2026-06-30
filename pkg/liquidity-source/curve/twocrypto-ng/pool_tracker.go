package twocryptong

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
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

	numTokens := len(p.Tokens)
	numDepCoins := numTokens - 1
	d := newRPCData(numTokens, numDepCoins)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy)
	addRPCCalls(func(c *ethrpc.Call, o []any) { calls.AddCall(c, o) }, p.Address, d)

	if res, err := calls.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	return buildPoolState(lg, p, d)
}

type rpcData struct {
	d, feeGamma, midFee, outFee, futureAGammaTime, futureAGamma, initialAGammaTime, initialAGamma *big.Int
	xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, lpSupply                         *big.Int
	math                                                                                           common.Address
	balances, priceScales, priceOracles, lastPrices                                                []*big.Int
}

func newRPCData(numTokens, numDepCoins int) *rpcData {
	return &rpcData{
		balances:     make([]*big.Int, numTokens),
		priceScales:  make([]*big.Int, numDepCoins),
		priceOracles: make([]*big.Int, numDepCoins),
		lastPrices:   make([]*big.Int, numDepCoins),
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, d *rpcData) {
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodD}, []any{&d.d})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodFeeGamma}, []any{&d.feeGamma})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodMidFee}, []any{&d.midFee})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodOutFee}, []any{&d.outFee})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodFutureAGammaTime}, []any{&d.futureAGammaTime})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodFutureAGamma}, []any{&d.futureAGamma})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodInitialAGammaTime}, []any{&d.initialAGammaTime})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodInitialAGamma}, []any{&d.initialAGamma})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodXcpProfit}, []any{&d.xcpProfit})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodVirtualPrice}, []any{&d.virtualPrice})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodAllowedExtraProfit}, []any{&d.allowedExtraProfit})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodAdjustmentStep}, []any{&d.adjustmentStep})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: shared.ERC20MethodTotalSupply}, []any{&d.lpSupply})
	addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodMath}, []any{&d.math})
	for i := range d.balances {
		addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodBalances, Params: []any{big.NewInt(int64(i))}}, []any{&d.balances[i]})
	}
	for i := range d.priceScales {
		addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodPriceScale}, []any{&d.priceScales[i]})
		addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodPriceOracle}, []any{&d.priceOracles[i]})
		addFn(&ethrpc.Call{ABI: curveTwocryptoNGABI, Target: poolAddress, Method: poolMethodLastPrices}, []any{&d.lastPrices[i]})
	}
}

func buildPoolState(lg logger.Logger, p entity.Pool, d *rpcData) (entity.Pool, error) {
	var extra = Extra{
		InitialA:           number.SetFromBig(new(big.Int).Rsh(d.initialAGamma, 128)),
		InitialGamma:       new(uint256.Int).And(number.SetFromBig(d.initialAGamma), PriceMask),
		InitialAGammaTime:  d.initialAGammaTime.Int64(),
		FutureA:            number.SetFromBig(new(big.Int).Rsh(d.futureAGamma, 128)),
		FutureGamma:        new(uint256.Int).And(number.SetFromBig(d.futureAGamma), PriceMask),
		FutureAGammaTime:   d.futureAGammaTime.Int64(),
		D:                  number.SetFromBig(d.d),
		FeeGamma:           number.SetFromBig(d.feeGamma),
		MidFee:             number.SetFromBig(d.midFee),
		OutFee:             number.SetFromBig(d.outFee),
		LpSupply:           number.SetFromBig(d.lpSupply),
		XcpProfit:          number.SetFromBig(d.xcpProfit),
		VirtualPrice:       number.SetFromBig(d.virtualPrice),
		AllowedExtraProfit: number.SetFromBig(d.allowedExtraProfit),
		AdjustmentStep:     number.SetFromBig(d.adjustmentStep),
		UseCustomMath:      UseCustomMath(d.math),
	}
	extra.PriceScale = make([]uint256.Int, len(d.priceScales))
	lo.ForEach(d.priceScales, func(item *big.Int, i int) { extra.PriceScale[i].SetFromBig(item) })
	extra.PriceOracle = make([]uint256.Int, len(d.priceOracles))
	lo.ForEach(d.priceOracles, func(item *big.Int, i int) { extra.PriceOracle[i].SetFromBig(item) })
	extra.LastPrices = make([]uint256.Int, len(d.lastPrices))
	lo.ForEach(d.lastPrices, func(item *big.Int, i int) { extra.LastPrices[i].SetFromBig(item) })

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(d.balances))
	for i := range d.balances {
		reserves = append(reserves, d.balances[i].String())
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}
