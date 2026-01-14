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

	// If we have oracle events, decode and update oracle rates
	// Oracle rate changes don't affect reserves but do affect pricing
	if hasOracleEvent {
		updatedPool, err := d.handleOracleEvents(ctx, p, params, extra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": p.Address,
				"error":       err,
			}).Warn("Failed to handle oracle events")
			// Continue with normal flow even if oracle update fails
		} else {
			return updatedPool, nil
		}
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

	// No relevant events, do a full refetch including oracle rates
	var reserves []*big.Int
	var updatedExtra Extra
	var err error

	// Check if we have oracle addresses to fetch rates
	if extra.BaseOracleAddress != "" || extra.QuoteOracleAddress != "" {
		// Fetch everything including oracle rates
		reserves, updatedExtra, err = d.fetchPoolStateWithOraclesFromNode(ctx, p.Address, extra.BaseOracleAddress, extra.QuoteOracleAddress)
	} else {
		// Fetch only reserves and curve params (no oracle addresses available yet)
		reserves, updatedExtra, err = d.fetchPoolStateFromNode(ctx, p.Address)
		// Preserve oracle addresses from existing extra if they exist
		updatedExtra.BaseOracleAddress = extra.BaseOracleAddress
		updatedExtra.QuoteOracleAddress = extra.QuoteOracleAddress
	}

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

// handleOracleEvents processes Chainlink oracle events (AnswerUpdated or NewTransmission)
// and updates the oracle rates in the pool state
func (d *PoolTracker) handleOracleEvents(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams, extra Extra) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("Processing oracle events")

	updatedExtra := extra
	hasBaseOracleUpdate := false
	hasQuoteOracleUpdate := false

	// Process oracle events
	for _, log := range params.Logs {
		// Check if it's from base oracle
		if extra.BaseOracleAddress != "" && isLogFromOracle(log, extra.BaseOracleAddress) {
			if isAnswerUpdatedEvent(log) {
				// AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt)
				// Topics: [0] = event sig, [1] = current (indexed), [2] = roundId (indexed)
				if len(log.Topics) >= 2 {
					// Extract current price from indexed topic
					currentPrice := new(big.Int).SetBytes(log.Topics[1].Bytes())
					updatedExtra.BaseOracleRate = currentPrice.String()
					hasBaseOracleUpdate = true
					logger.WithFields(logger.Fields{
						"dex":         DexType,
						"poolAddress": p.Address,
						"oracle":      "base",
						"rate":        currentPrice.String(),
					}).Info("Updated base oracle rate from AnswerUpdated event")
				}
			} else if isNewTransmissionEvent(log) {
				// NewTransmission(uint32 indexed aggregatorRoundId, int192 answer, ...)
				// Decode answer from log.Data
				if len(log.Data) == 0 {
					logger.WithFields(logger.Fields{
						"dex":    DexType,
						"oracle": "base",
					}).Warn("NewTransmission event has empty data, skipping")
					continue
				}
				type NewTransmissionEvent struct {
					Answer *big.Int
				}
				var event NewTransmissionEvent
				if err := chainlinkAggregatorABI.UnpackIntoInterface(&event, "NewTransmission", log.Data); err != nil {
					logger.WithFields(logger.Fields{
						"dex":   DexType,
						"error": err,
					}).Warn("Failed to decode NewTransmission event for base oracle")
					continue
				}
				updatedExtra.BaseOracleRate = event.Answer.String()
				hasBaseOracleUpdate = true
				logger.WithFields(logger.Fields{
					"dex":         DexType,
					"poolAddress": p.Address,
					"oracle":      "base",
					"rate":        event.Answer.String(),
				}).Info("Updated base oracle rate from NewTransmission event")
			}
		}

		// Check if it's from quote oracle
		if extra.QuoteOracleAddress != "" && isLogFromOracle(log, extra.QuoteOracleAddress) {
			if isAnswerUpdatedEvent(log) {
				// AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt)
				if len(log.Topics) >= 2 {
					currentPrice := new(big.Int).SetBytes(log.Topics[1].Bytes())
					updatedExtra.QuoteOracleRate = currentPrice.String()
					hasQuoteOracleUpdate = true
					logger.WithFields(logger.Fields{
						"dex":         DexType,
						"poolAddress": p.Address,
						"oracle":      "quote",
						"rate":        currentPrice.String(),
					}).Info("Updated quote oracle rate from AnswerUpdated event")
				}
			} else if isNewTransmissionEvent(log) {
				if len(log.Data) == 0 {
					logger.WithFields(logger.Fields{
						"dex":    DexType,
						"oracle": "quote",
					}).Warn("NewTransmission event has empty data, skipping")
					continue
				}
				type NewTransmissionEvent struct {
					Answer *big.Int
				}
				var event NewTransmissionEvent
				if err := chainlinkAggregatorABI.UnpackIntoInterface(&event, "NewTransmission", log.Data); err != nil {
					logger.WithFields(logger.Fields{
						"dex":   DexType,
						"error": err,
					}).Warn("Failed to decode NewTransmission event for quote oracle")
					continue
				}
				updatedExtra.QuoteOracleRate = event.Answer.String()
				hasQuoteOracleUpdate = true
				logger.WithFields(logger.Fields{
					"dex":         DexType,
					"poolAddress": p.Address,
					"oracle":      "quote",
					"rate":        event.Answer.String(),
				}).Info("Updated quote oracle rate from NewTransmission event")
			}
		}
	}

	if !hasBaseOracleUpdate && !hasQuoteOracleUpdate {
		return p, fmt.Errorf("no oracle events decoded")
	}

	// Recalculate derived oracle rate if both rates are available
	if updatedExtra.BaseOracleRate != "" && updatedExtra.QuoteOracleRate != "" {
		baseRate, ok1 := new(big.Int).SetString(updatedExtra.BaseOracleRate, 10)
		quoteRate, ok2 := new(big.Int).SetString(updatedExtra.QuoteOracleRate, 10)
		if ok1 && ok2 && quoteRate.Cmp(big.NewInt(0)) > 0 {
			// oracleRate = baseRate / quoteRate (scaled by precision)
			oracleRate := new(big.Int).Mul(baseRate, BigOne)
			oracleRate.Div(oracleRate, quoteRate)
			updatedExtra.OracleRate = oracleRate.String()
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
		"baseRate":    updatedExtra.BaseOracleRate,
		"quoteRate":   updatedExtra.QuoteOracleRate,
		"derivedRate": updatedExtra.OracleRate,
	}).Info("Successfully updated oracle rates from events")

	return p, nil
}

// fetchPoolStateFromNode fetches current reserves, curve parameters, and oracle rates from the blockchain
func (d *PoolTracker) fetchPoolStateFromNode(ctx context.Context, poolAddress string) ([]*big.Int, Extra, error) {
	// First, get the current Extra to retrieve oracle addresses
	// We need to fetch this from the pool entity stored in state
	// For now, we'll fetch everything fresh and populate oracle addresses later

	// Define struct to match the liquidity() return signature
	type LiquidityResult struct {
		Total      *big.Int
		Individual []*big.Int
	}

	// Define struct to match viewCurve() return signature
	type CurveResult struct {
		Alpha   *big.Int
		Beta    *big.Int
		Delta   *big.Int
		Epsilon *big.Int
		Lambda  *big.Int
	}

	var (
		liquidityResult LiquidityResult
		curveResult     CurveResult
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult})

	// Fetch curve parameters using viewCurve() method
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult})

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

	// Build extra without oracle data (will be populated by fetchOracleRates)
	extra := Extra{
		CurveParams: curveParams,
	}

	return reserves, extra, nil
}

// fetchPoolStateWithOraclesFromNode fetches reserves, curve parameters, AND oracle rates from the blockchain
func (d *PoolTracker) fetchPoolStateWithOraclesFromNode(ctx context.Context, poolAddress string, baseOracleAddr, quoteOracleAddr string) ([]*big.Int, Extra, error) {
	// Define struct to match the liquidity() return signature
	type LiquidityResult struct {
		Total      *big.Int
		Individual []*big.Int
	}

	// Define struct to match viewCurve() return signature
	type CurveResult struct {
		Alpha   *big.Int
		Beta    *big.Int
		Delta   *big.Int
		Epsilon *big.Int
		Lambda  *big.Int
	}

	var (
		liquidityResult LiquidityResult
		curveResult     CurveResult
		baseOracleRate  *big.Int
		quoteOracleRate *big.Int
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult})

	// Fetch curve parameters using viewCurve() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult})

	// Fetch base oracle rate (if address provided)
	if baseOracleAddr != "" {
		baseOracleRate = new(big.Int)
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: baseOracleAddr,
			Method: oracleMethodLatestAnswer,
			Params: []interface{}{},
		}, []interface{}{&baseOracleRate})
	}

	// Fetch quote oracle rate (if address provided)
	if quoteOracleAddr != "" {
		quoteOracleRate = new(big.Int)
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: quoteOracleAddr,
			Method: oracleMethodLatestAnswer,
			Params: []interface{}{},
		}, []interface{}{&quoteOracleRate})
	}

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, Extra{}, fmt.Errorf("failed to fetch pool state with oracles: %w", err)
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

	// Build extra with oracle data
	extra := Extra{
		CurveParams:        curveParams,
		BaseOracleAddress:  baseOracleAddr,
		QuoteOracleAddress: quoteOracleAddr,
	}

	// Set oracle rates if fetched
	if baseOracleRate != nil {
		extra.BaseOracleRate = baseOracleRate.String()
	}
	if quoteOracleRate != nil {
		extra.QuoteOracleRate = quoteOracleRate.String()
	}

	// Calculate derived oracle rate if both rates are available
	if baseOracleRate != nil && quoteOracleRate != nil && quoteOracleRate.Cmp(big.NewInt(0)) > 0 {
		// oracleRate = baseRate / quoteRate (scaled by precision)
		oracleRate := new(big.Int).Mul(baseOracleRate, BigOne)
		oracleRate.Div(oracleRate, quoteOracleRate)
		extra.OracleRate = oracleRate.String()
	}

	return reserves, extra, nil
}

// fetchPoolReservesFromNode fetches only reserves (and optionally oracle rates) - efficient for Trade events
func (d *PoolTracker) fetchPoolReservesFromNode(ctx context.Context, poolAddress string, existingExtra Extra) ([]*big.Int, Extra, error) {
	// Define struct to match the liquidity() return signature
	type LiquidityResult struct {
		Total      *big.Int
		Individual []*big.Int
	}

	var (
		liquidityResult LiquidityResult
		baseOracleRate  *big.Int
		quoteOracleRate *big.Int
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult})

	// Also fetch oracle rates if addresses are available (good to refresh periodically)
	if existingExtra.BaseOracleAddress != "" {
		baseOracleRate = new(big.Int)
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: existingExtra.BaseOracleAddress,
			Method: oracleMethodLatestAnswer,
			Params: []interface{}{},
		}, []interface{}{&baseOracleRate})
	}

	if existingExtra.QuoteOracleAddress != "" {
		quoteOracleRate = new(big.Int)
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: existingExtra.QuoteOracleAddress,
			Method: oracleMethodLatestAnswer,
			Params: []interface{}{},
		}, []interface{}{&quoteOracleRate})
	}

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, Extra{}, err
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return nil, Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Update oracle rates if fetched
	if baseOracleRate != nil {
		existingExtra.BaseOracleRate = baseOracleRate.String()
	}
	if quoteOracleRate != nil {
		existingExtra.QuoteOracleRate = quoteOracleRate.String()
	}

	// Recalculate derived oracle rate if both rates are available
	if baseOracleRate != nil && quoteOracleRate != nil && quoteOracleRate.Cmp(big.NewInt(0)) > 0 {
		oracleRate := new(big.Int).Mul(baseOracleRate, BigOne)
		oracleRate.Div(oracleRate, quoteOracleRate)
		existingExtra.OracleRate = oracleRate.String()
	}

	// Return existing extra with updated oracle rates (curve parameters unchanged)
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

		// Check if log data is empty
		if len(log.Data) == 0 {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": p.Address,
			}).Warn("ParametersSet event has empty data, skipping")
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
