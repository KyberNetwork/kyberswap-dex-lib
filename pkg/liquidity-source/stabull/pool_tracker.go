package stabull

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

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

	// Check if we have Trade events to process
	// If we do, we can update reserves from the event data instead of fetching from RPC
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Warn("Failed to decode extra data, will refetch from node")
	}

	hasTradeEvent := false
	hasParametersSetEvent := false
	hasOracleEvent := false

	// Process events if provided
	for _, log := range params.Logs {
		if !isLogFromPool(log, p.Address) {
			// Check if it's from one of the oracle contracts
			if extra.BaseOracleAddress != "" && isLogFromOracle(log, extra.BaseOracleAddress) {
				hasOracleEvent = true
			} else if extra.QuoteOracleAddress != "" && isLogFromOracle(log, extra.QuoteOracleAddress) {
				hasOracleEvent = true
			}
			continue
		}

		if isTradeEvent(log) {
			hasTradeEvent = true
		} else if isParametersSetEvent(log) {
			hasParametersSetEvent = true
		}
	}

	// If we have a ParametersSet event, we need to update curve parameters
	if hasParametersSetEvent {
		return d.handleParametersSetEvent(ctx, p, params)
	}

	// If we have oracle events, refetch oracle rates
	// Oracle rate changes don't affect reserves but do affect pricing
	if hasOracleEvent {
		// Oracle updates are less critical for reserve tracking
		// We can optionally fetch new oracle rates here
		// For now, we'll let the next regular update handle it
	}

	// For Trade events, we can either:
	// 1. Decode the event and update reserves directly (more efficient)
	// 2. Refetch reserves from node (simpler, what we're doing now)
	//
	// Since Trade event includes originAmount and targetAmount, we could update reserves:
	// - reserves[origin] += originAmount
	// - reserves[target] -= targetAmount
	//
	// However, for simplicity and to ensure accuracy, we refetch from node
	if hasTradeEvent {
		// Refetch reserves after trade
		reserves, updatedExtra, err := d.fetchPoolReservesFromNode(ctx, p.Address, extra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": p.Address,
				"error":       err,
			}).Error("Failed to fetch pool reserves after trade event")
			return p, err
		}

		// Update reserves
		for i := range p.Reserves {
			if i < len(reserves) {
				p.Reserves[i] = reserves[i].String()
			}
		}

		// Update extra data
		extraBytes, err := json.Marshal(updatedExtra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
		p.Timestamp = time.Now().Unix()

		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
		}).Info("Updated reserves after trade event")

		return p, nil
	}

	// No relevant events, do a full refetch
	reserves, updatedExtra, err := d.fetchPoolStateFromNode(ctx, p.Address)
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
	extraBytes, err := json.Marshal(updatedExtra)
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

	// Get oracle addresses from config if available
	extra := Extra{
		CurveParams: curveParams,
	}

	// Optionally populate oracle addresses from config
	// This would be set during pool initialization from d.config.ChainlinkOracles

	return reserves, extra, nil
}

// fetchPoolReservesFromNode fetches only reserves (not curve parameters) - more efficient for Trade events
func (d *PoolTracker) fetchPoolReservesFromNode(ctx context.Context, poolAddress string, existingExtra Extra) ([]*big.Int, Extra, error) {
	var liquidityResult struct {
		Total      *big.Int   `json:"total_"`
		Individual []*big.Int `json:"individual_"`
	}

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult.Total, &liquidityResult.Individual})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, Extra{}, err
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return nil, Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Return existing extra (curve parameters unchanged)
	return reserves, existingExtra, nil
}

// handleParametersSetEvent processes a ParametersSet event by decoding the event data
// ParametersSet(uint256 alpha, uint256 beta, uint256 delta, uint256 epsilon, uint256 lambda)
// This is emitted when admin updates curve parameters - infrequent but critical for pricing
func (d *PoolTracker) handleParametersSetEvent(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("ParametersSet event detected, decoding event data")

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Warn("Failed to decode existing extra data")
	}

	// Find and decode the ParametersSet event from the logs
	for _, log := range params.Logs {
		if !isLogFromPool(log, p.Address) || !isParametersSetEvent(log) {
			continue
		}

		// Decode event data
		type ParametersSetEvent struct {
			Alpha   *big.Int
			Beta    *big.Int
			Delta   *big.Int
			Epsilon *big.Int
			Lambda  *big.Int
		}
		var event ParametersSetEvent
		if err := stabullPoolABI.UnpackIntoInterface(&event, "ParametersSet", log.Data); err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": p.Address,
				"error":       err,
			}).Error("Failed to decode ParametersSet event")
			continue
		}

		// Update curve parameters in extra (convert to strings)
		extra.CurveParams = CurveParameters{
			Alpha:   event.Alpha.String(),
			Beta:    event.Beta.String(),
			Delta:   event.Delta.String(),
			Epsilon: event.Epsilon.String(),
			Lambda:  event.Lambda.String(),
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
		p.Timestamp = time.Now().Unix()

		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"alpha":       event.Alpha.String(),
			"beta":        event.Beta.String(),
			"delta":       event.Delta.String(),
			"epsilon":     event.Epsilon.String(),
			"lambda":      event.Lambda.String(),
		}).Info("Updated curve parameters from ParametersSet event")

		return p, nil
	}

	// If we couldn't find/decode the event, fall back to RPC call
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Warn("No ParametersSet event found in logs, falling back to RPC")

	reserves, updatedExtra, err := d.fetchPoolStateFromNode(ctx, p.Address)
	if err != nil {
		return p, err
	}

	// Update reserves
	for i := range p.Reserves {
		if i < len(reserves) {
			p.Reserves[i] = reserves[i].String()
		}
	}

	extraBytes, err := json.Marshal(updatedExtra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
