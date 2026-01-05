package stabull

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

// GetNewPoolState updates the pool state by fetching current reserves and parameters
func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("Start getting new state of pool")

	// Fetch current pool state from blockchain
	reserves, extra, err := d.fetchPoolStateFromNode(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Error("Failed to fetch pool state")
		return p, err
	}

	// Update reserves
	for i := range p.Reserves {
		if i < len(reserves) {
			p.Reserves[i] = reserves[i].String()
		}
	}

	// Update extra data
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("Finished getting new state of pool")

	return p, nil
}

// fetchPoolStateFromNode fetches current reserves and curve parameters from the blockchain
func (d *PoolTracker) fetchPoolStateFromNode(ctx context.Context, poolAddress string) ([]*big.Int, Extra, error) {
	var (
		liquidityResult struct {
			Total      *big.Int   `json:"total_"`
			Individual []*big.Int `json:"individual_"`
		}
		curveResult struct {
			Alpha   *big.Int `json:"alpha_"`
			Beta    *big.Int `json:"beta_"`
			Delta   *big.Int `json:"delta_"`
			Epsilon *big.Int `json:"epsilon_"`
			Lambda  *big.Int `json:"lambda_"`
		}
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult.Total, &liquidityResult.Individual})

	// Fetch curve parameters using viewCurve() method
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult.Alpha, &curveResult.Beta, &curveResult.Delta, &curveResult.Epsilon, &curveResult.Lambda})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, Extra{}, err
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return nil, Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParameters{
		Alpha:   curveResult.Alpha.String(),
		Beta:    curveResult.Beta.String(),
		Delta:   curveResult.Delta.String(),
		Epsilon: curveResult.Epsilon.String(),
		Lambda:  curveResult.Lambda.String(),
	}

	// Oracle rates: Stabull uses Chainlink oracles internally
	// Oracle rates: Stabull uses Chainlink oracles internally
	// The viewOriginSwap already accounts for oracle-adjusted pricing
	// We don't need to fetch oracle rates separately for simulation

	extra := Extra{
		CurveParams:     curveParams,
		BaseOracleRate:  "", // Not needed - included in viewOriginSwap
		QuoteOracleRate: "", // Not needed - included in viewOriginSwap
		OracleRate:      "",
	}

	return reserves, extra, nil
}

// processLog processes different event types (called internally during GetNewPoolState)
func (d *PoolTracker) processLog(ctx context.Context, log types.Log, pool entity.Pool) (entity.Pool, error) {
	eventSignature := log.Topics[0].Hex()

	switch eventSignature {
	case tradeEventTopic:
		return d.handleTradeEvent(ctx, log, pool)
	case parametersSetEventTopic:
		return d.handleParametersSetEvent(ctx, log, pool)
	default:
		// Unknown event, return pool unchanged
		return pool, nil
	}
}

// handleTradeEvent processes a Trade event (swap transaction)
// Trade(address indexed trader, address indexed origin, address indexed target, uint256 originAmount, uint256 targetAmount, int128 rawProtocolFee)
// Note: We refetch reserves as Trade events affect pool balances
func (d *PoolTracker) handleTradeEvent(ctx context.Context, log types.Log, p entity.Pool) (entity.Pool, error) {
	// Trade events change reserves, so refetch from node
	return d.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: []types.Log{log}})
}

// handleParametersSetEvent processes a ParametersSet event
// ParametersSet(uint256 alpha, uint256 beta, uint256 delta, uint256 epsilon, uint256 lambda)
// This is emitted when admin updates curve parameters - infrequent but critical for pricing
func (d *PoolTracker) handleParametersSetEvent(ctx context.Context, log types.Log, pool entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": pool.Address,
	}).Info("ParameterSet event detected, refetching pool state")

	// ParameterSet changes the curve parameters, so we need to refetch viewCurve()
	// Reserves don't change on ParameterSet, but pricing formula does

	var extra Extra
	if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": pool.Address,
			"error":       err,
		}).Warn("Failed to decode existing extra data")
	}

	// Fetch updated curve parameters
	var curveResult struct {
		Alpha   *big.Int
		Beta    *big.Int
		Delta   *big.Int
		Epsilon *big.Int
		Lambda  *big.Int
	}
	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: pool.Address,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult.Alpha, &curveResult.Beta, &curveResult.Delta, &curveResult.Epsilon, &curveResult.Lambda})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": pool.Address,
			"error":       err,
		}).Error("Failed to fetch updated curve parameters")
		return pool, err
	}

	// Update curve parameters in extra (convert to strings)
	extra.CurveParams = CurveParameters{
		Alpha:   curveResult.Alpha.String(),
		Beta:    curveResult.Beta.String(),
		Delta:   curveResult.Delta.String(),
		Epsilon: curveResult.Epsilon.String(),
		Lambda:  curveResult.Lambda.String(),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}
	pool.Extra = string(extraBytes)
	pool.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": pool.Address,
		"alpha":       curveResult.Alpha.String(),
		"beta":        curveResult.Beta.String(),
		"delta":       curveResult.Delta.String(),
		"epsilon":     curveResult.Epsilon.String(),
		"lambda":      curveResult.Lambda.String(),
	}).Info("Updated curve parameters after ParametersSet event")

	return pool, nil
}
