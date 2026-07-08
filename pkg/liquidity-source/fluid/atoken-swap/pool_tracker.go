package atokenswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	extra, blockNumber, err := t.getPoolState(ctx, &p, overrides)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("error marshaling extra data")
		return p, err
	}

	// Update reserves based on available liquidity for all output tokens
	reserves := make(entity.PoolReserves, len(extra.OutputStates)+1)
	reserves[0] = "0" // Input token reserve (not tracked)
	for i, state := range extra.OutputStates {
		reserves[i+1] = state.AvailableLiquidity.String()
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) getPoolState(
	ctx context.Context,
	p *entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	// Prepare variables for all output tokens
	var paused bool
	var premium, oraclePrecision *big.Int
	rateVars := make([]*big.Int, len(p.Tokens)-1)
	liquidityVars := make([]*big.Int, len(p.Tokens)-1)
	maxSwapVars := make([]*big.Int, len(p.Tokens)-1)

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: p.Address,
		Method: "paused",
	}, []any{&paused}).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: p.Address,
		Method: "premium",
	}, []any{&premium}).AddCall(&ethrpc.Call{
		ABI:    aTokenSwapABI,
		Target: p.Address,
		Method: "ORACLE_PRECISION",
	}, []any{&oraclePrecision})

	prefixLen := len(p.Tokens[0].Symbol) - 4 // aEthWETH -> aEth
	for i, token := range p.Tokens[1:] {
		// aEthwstETH - aEth -> wstETH -> WstETH
		shortSymbol := strings.ToUpper(token.Symbol[prefixLen:prefixLen+1]) + token.Symbol[prefixLen+1:]
		rateFunc := "get" + shortSymbol + "Rate"
		liquidityFunc := "available" + shortSymbol + "Liquidity"
		maxSwapFunc := "maxSwapTo" + shortSymbol

		rateVars[i] = new(big.Int)
		liquidityVars[i] = new(big.Int)
		maxSwapVars[i] = new(big.Int)

		req = req.AddCall(&ethrpc.Call{
			ABI:    aTokenSwapABI,
			Target: p.Address,
			Method: rateFunc,
		}, []any{&rateVars[i]}).AddCall(&ethrpc.Call{
			ABI:    aTokenSwapABI,
			Target: p.Address,
			Method: liquidityFunc,
		}, []any{&liquidityVars[i]}).AddCall(&ethrpc.Call{
			ABI:    aTokenSwapABI,
			Target: p.Address,
			Method: maxSwapFunc,
		}, []any{&maxSwapVars[i]})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("failed to get pool state")
		return Extra{}, 0, err
	}

	// Build output states - map all tokens (zero values for unsupported ones)
	var tmp big.Int
	tmp.Sub(PremiumPrecision, premium)
	outputStates := lo.Map(rateVars, func(rate *big.Int, i int) OutputTokenState {
		rate = bignumber.MulDivDown(rate, rate, PremiumPrecision, &tmp)
		if oraclePrecision != nil {
			rate = bignumber.MulDivDown(rate, rate, bignumber.BONE, oraclePrecision)
		}
		return OutputTokenState{
			RateWithPremium:    uint256.MustFromBig(rate),
			AvailableLiquidity: uint256.MustFromBig(liquidityVars[i]),
			MaxSwap:            uint256.MustFromBig(maxSwapVars[i]),
		}
	})

	return Extra{
		Paused:       paused,
		OutputStates: outputStates,
	}, resp.BlockNumber.Uint64(), nil
}
