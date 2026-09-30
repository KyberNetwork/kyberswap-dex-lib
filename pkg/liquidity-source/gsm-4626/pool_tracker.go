package gsm4626

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("start get new state %v", p.Address)
	defer func() {
		logger.Infof("finish get new pool state %v", p.Address)
	}()

	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)

	var (
		canSwap         bool
		currentExposure *uint256.Int
		exposureCap     *uint256.Int
		rate            *uint256.Int
		feeStrategy     common.Address
		tokenBalance    *uint256.Int
		ghoUsage        struct {
			Limit *uint256.Int `abi:"limit"`
			Used  *uint256.Int `abi:"used"`
		}
	)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodCanSwap,
		}, []any{&canSwap}).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetAvailableLiquidity,
		}, []any{&currentExposure}).
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetExposureCap,
		}, []any{&exposureCap}).
		AddCall(&ethrpc.Call{
			ABI:    erc4626.ABI,
			Target: p.Tokens[1].Address,
			Method: erc4626.Erc4626MethodConvertToAssets,
			Params: []any{ray.ToBig()},
		}, []any{&rate}). // convertToAssets(amt) = amt * rate() / ray
		AddCall(&ethrpc.Call{
			ABI:    gsm4626ABI,
			Target: p.Address,
			Method: gsmMethodGetFeeStrategy,
		}, []any{&feeStrategy}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{staticExtra.GhoReserve},
		}, []any{&tokenBalance}).
		AddCall(&ethrpc.Call{
			ABI:    ghoReserveABI,
			Target: staticExtra.GhoReserve.String(),
			Method: ghoReserveMethodGetUsage,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&ghoUsage}).
		Aggregate()
	if err != nil {
		return p, err
	}

	var (
		sellFee *uint256.Int
		buyFee  *uint256.Int
	)
	if !eth.IsZeroAddress(feeStrategy) {
		if _, err = t.ethrpcClient.NewRequest().SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    feeStrategyABI,
				Target: feeStrategy.String(),
				Method: feeStrategyMethodGetSellFee,
				Params: []any{percentageFactor.ToBig()},
			}, []any{&sellFee}).
			AddCall(&ethrpc.Call{
				ABI:    feeStrategyABI,
				Target: feeStrategy.String(),
				Method: feeStrategyMethodGetBuyFee,
				Params: []any{percentageFactor.ToBig()},
			}, []any{&buyFee}).
			Aggregate(); err != nil {
			return p, nil
		}
	} else {
		sellFee, buyFee = new(uint256.Int), new(uint256.Int)
	}

	extraBytes, err := json.Marshal(Extra{
		CanSwap:         canSwap,
		CurrentExposure: cloneOrZero(currentExposure),
		ExposureCap:     cloneOrZero(exposureCap),
		Rate:            cloneOrZero(rate),
		BuyFee:          cloneOrZero(buyFee),
		SellFee:         cloneOrZero(sellFee),
		GhoLimit:        cloneOrZero(ghoUsage.Limit),
		GhoUsed:         cloneOrZero(ghoUsage.Used),
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	// tokenBalance is GhoReserve's total balance shared across all its facilitators;
	// this pool can only ever draw up to its own remaining bucket (ghoLimit - ghoUsed).
	availableGho := new(uint256.Int)
	if cloneOrZero(ghoUsage.Limit).Cmp(cloneOrZero(ghoUsage.Used)) >= 0 {
		availableGho.Sub(cloneOrZero(ghoUsage.Limit), cloneOrZero(ghoUsage.Used))
	}
	ghoReserve := availableGho
	if cloneOrZero(tokenBalance).Lt(availableGho) {
		ghoReserve = tokenBalance
	}
	p.Reserves = []string{cloneOrZero(ghoReserve).Dec(), cloneOrZero(currentExposure).Dec()}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	return p, nil
}

func cloneOrZero(x *uint256.Int) *uint256.Int {
	if x == nil {
		return new(uint256.Int)
	}

	return new(uint256.Int).Set(x)
}
