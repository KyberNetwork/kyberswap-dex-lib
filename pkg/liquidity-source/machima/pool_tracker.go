package machima

import (
	"context"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexTypeMachima, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	_ *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams) (entity.Pool, error) {
	// Phase 1: Read pool state (slot0 + liquidity)
	var (
		slot0Result Slot0
		liquidityBI *big.Int
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "slot0",
		Params: nil,
	}, []interface{}{&slot0Result})
	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "liquidity",
		Params: nil,
	}, []interface{}{&liquidityBI})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Address,
			"error": err,
		}).Error("failed to get machima pool state")
		return entity.Pool{}, err
	}

	// Phase 2: Decode static extra
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	counterAsset := staticExtra.CounterAsset
	token := staticExtra.Token

	// Phase 3: Read TaxConfig from ClankNow (full struct decode).
	// ethrpc/go-ethereum requires a wrapper struct for tuple outputs.
	var taxResult struct{ Data TaxConfig }
	taxCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
	taxCalls.AddCall(&ethrpc.Call{
		ABI:    clankNowABI,
		Target: t.config.ClankNow,
		Method: "getTokenTax",
		Params: []interface{}{common.HexToAddress(token)},
	}, []interface{}{&taxResult})

	var taxConfig TaxConfig
	if _, err := taxCalls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Address,
			"token": token,
			"error": err,
		}).Warn("failed to get token tax config, defaulting to no tax")
	} else {
		taxConfig = taxResult.Data
	}

	// Phase 3b: Read xmaSellSqrtPriceLimit from the swap adapter.
	// Only relevant when the token being sold is XMA.
	var xmaFloorBI *big.Int
	if t.config.SwapAdapter != "" {
		floorCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
		floorCalls.AddCall(&ethrpc.Call{
			ABI:    swapAdapterABI,
			Target: t.config.SwapAdapter,
			Method: "xmaSellSqrtPriceLimit",
			Params: nil,
		}, []interface{}{&xmaFloorBI})

		if _, err := floorCalls.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"pool":  p.Address,
				"error": err,
			}).Warn("failed to get xmaSellSqrtPriceLimit, defaulting to 0")
			xmaFloorBI = nil
		}
	}

	// Phase 4: Read poolDeploymentTime from the token contract
	var poolDeploymentTime *big.Int
	tokenCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
	tokenCalls.AddCall(&ethrpc.Call{
		ABI:    tokenABI,
		Target: token,
		Method: "poolDeploymentTime",
		Params: nil,
	}, []interface{}{&poolDeploymentTime})

	if _, err := tokenCalls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Address,
			"token": token,
			"error": err,
		}).Warn("failed to get poolDeploymentTime, using entity timestamp")
		poolDeploymentTime = big.NewInt(p.Timestamp)
	}

	// Phase 5: Fetch ticks via TickLens.
	// Seed p.Extra with tickSpacing so ticklens can read it on the first cycle
	// (otherwise unmarshal of empty Extra fails and returns no ticks).
	if p.Extra == "" {
		seedExtra, _ := json.Marshal(Extra{TickSpacing: TickSpacing})
		p.Extra = string(seedExtra)
	}
	tickResps, err := ticklens.GetPoolTicksFromSC(ctx, t.ethrpcClient, t.config.TickLensAddress, p, params.Logs)
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Address,
			"error": err,
		}).Warn("failed to fetch ticks via TickLens, using empty ticks")
		tickResps = nil
	}

	// Convert ticks to our TickData format.
	// int256.Int and uint256.Int share JSON representation (decimal strings),
	// so FromBig works for both positive and negative values via two's complement.
	tickData := make([]TickData, 0, len(tickResps))
	for _, tr := range tickResps {
		idx, _ := strconv.Atoi(tr.TickIdx)
		lg, ok1 := new(big.Int).SetString(tr.LiquidityGross, 10)
		ln, ok2 := new(big.Int).SetString(tr.LiquidityNet, 10)
		if !ok1 || !ok2 {
			continue
		}

		liquidityGross, _ := uint256.FromBig(lg)
		liquidityNet := new(int256.Int)
		liquidityNet.SetFromBig(ln)

		tickData = append(tickData, TickData{
			Index:          idx,
			LiquidityGross: liquidityGross,
			LiquidityNet:   liquidityNet,
		})
	}

	// Build extra
	sqrtPriceX96, _ := uint256.FromBig(slot0Result.SqrtPriceX96)
	liquidity, _ := uint256.FromBig(liquidityBI)
	tick := int(slot0Result.Tick.Int64())

	var xmaFloor *uint256.Int
	if xmaFloorBI != nil && xmaFloorBI.Sign() > 0 {
		xmaFloor, _ = uint256.FromBig(xmaFloorBI)
	}

	extra := Extra{
		SqrtPriceX96:          sqrtPriceX96,
		Tick:                  &tick,
		Liquidity:             liquidity,
		TickSpacing:           TickSpacing,
		Ticks:                 tickData,
		BuyTaxBps:             taxConfig.BuyTaxBps,
		SellTaxBps:            taxConfig.SellTaxBps,
		HasTax:                taxConfig.HasTax,
		CounterAsset:          counterAsset,
		Token:                 token,
		PoolDeploymentTime:    poolDeploymentTime.Uint64(),
		XmaSellSqrtPriceLimit: xmaFloor,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)

	// Phase 7: Update reserves from on-chain token balances
	if len(p.Tokens) == 2 {
		reserves := make([]string, 2)
		resCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
		var bal0, bal1 *big.Int
		resCalls.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: "balanceOf",
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&bal0})
		resCalls.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: "balanceOf",
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&bal1})

		if _, err := resCalls.Aggregate(); err == nil {
			reserves[0] = bal0.String()
			reserves[1] = bal1.String()
			p.Reserves = reserves
		}
	}

	return p, nil
}

// Slot0 mirrors UniV3 slot0 return values
type Slot0 struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint8
	Unlocked                   bool
}
