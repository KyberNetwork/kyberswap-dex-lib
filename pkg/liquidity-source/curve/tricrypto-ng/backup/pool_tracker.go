package backup

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
	tricryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

const (
	poolMethodD                  = "D"
	poolMethodFeeGamma           = "fee_gamma"
	poolMethodMidFee             = "mid_fee"
	poolMethodOutFee             = "out_fee"
	poolMethodInitialAGamma      = "initial_A_gamma"
	poolMethodInitialAGammaTime  = "initial_A_gamma_time"
	poolMethodFutureAGamma       = "future_A_gamma"
	poolMethodFutureAGammaTime   = "future_A_gamma_time"
	poolMethodXcpProfit          = "xcp_profit"
	poolMethodVirtualPrice       = "virtual_price"
	poolMethodAllowedExtraProfit = "allowed_extra_profit"
	poolMethodAdjustmentStep     = "adjustment_step"
	poolMethodBalances           = "balances"
	poolMethodPriceScale         = "price_scale"
	poolMethodPriceOracle        = "price_oracle"
	poolMethodLastPrices         = "last_prices"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterBackupFactoryCE(tricryptong.DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": tricryptong.DexType,
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

		xcpProfit, virtualPrice, allowedExtraProfit, adjustmentStep, lpSupply *big.Int

		balances = make([]*big.Int, len(p.Tokens))

		numDepCoins = len(p.Tokens) - 1

		priceScales  = make([]*big.Int, numDepCoins)
		priceOracles = make([]*big.Int, numDepCoins)
		lastPrices   = make([]*big.Int, numDepCoins)
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy).
		AddCall(&ethrpc.Call{
			ABI:    *tricryptong.CurveTricryptoNGABI,
			Target: p.Address,
			Method: poolMethodD,
		}, []any{&d}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodFeeGamma,
	}, []any{&feeGamma}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodMidFee,
	}, []any{&midFee}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodOutFee,
	}, []any{&outFee}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodFutureAGammaTime,
	}, []any{&futureAGammaTime}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodFutureAGamma,
	}, []any{&futureAGamma}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodInitialAGammaTime,
	}, []any{&initialAGammaTime}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodInitialAGamma,
	}, []any{&initialAGamma}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodXcpProfit,
	}, []any{&xcpProfit}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodVirtualPrice,
	}, []any{&virtualPrice}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodAllowedExtraProfit,
	}, []any{&allowedExtraProfit}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: poolMethodAdjustmentStep,
	}, []any{&adjustmentStep}).AddCall(&ethrpc.Call{
		ABI:    *tricryptong.CurveTricryptoNGABI,
		Target: p.Address,
		Method: shared.ERC20MethodTotalSupply,
	}, []any{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    *tricryptong.CurveTricryptoNGABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&balances[i]})
	}

	for i := range numDepCoins {
		calls.AddCall(&ethrpc.Call{
			ABI:    *tricryptong.CurveTricryptoNGABI,
			Target: p.Address,
			Method: poolMethodPriceScale,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&priceScales[i]}).AddCall(&ethrpc.Call{
			ABI:    *tricryptong.CurveTricryptoNGABI,
			Target: p.Address,
			Method: poolMethodPriceOracle,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&priceOracles[i]}).AddCall(&ethrpc.Call{
			ABI:    *tricryptong.CurveTricryptoNGABI,
			Target: p.Address,
			Method: poolMethodLastPrices,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&lastPrices[i]})
	}
	if res, err := calls.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	var extra = tricryptong.Extra{
		InitialA:           number.SetFromBig(new(big.Int).Rsh(initialAGamma, 128)),
		InitialGamma:       new(uint256.Int).And(number.SetFromBig(initialAGamma), tricryptong.PriceMask),
		InitialAGammaTime:  initialAGammaTime.Int64(),
		FutureA:            number.SetFromBig(new(big.Int).Rsh(futureAGamma, 128)),
		FutureGamma:        new(uint256.Int).And(number.SetFromBig(futureAGamma), tricryptong.PriceMask),
		FutureAGammaTime:   futureAGammaTime.Int64(),
		D:                  number.SetFromBig(d),
		FeeGamma:           number.SetFromBig(feeGamma),
		MidFee:             number.SetFromBig(midFee),
		OutFee:             number.SetFromBig(outFee),
		LpSupply:           number.SetFromBig(lpSupply),
		XcpProfit:          number.SetFromBig(xcpProfit),
		VirtualPrice:       number.SetFromBig(virtualPrice),
		AllowedExtraProfit: number.SetFromBig(allowedExtraProfit),
		AdjustmentStep:     number.SetFromBig(adjustmentStep),
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
