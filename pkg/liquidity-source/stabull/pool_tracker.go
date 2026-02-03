package stabull

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

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
func (t *PoolTracker) GetNewPoolState(
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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Error("failed to decode StaticExtra data")
		return entity.Pool{}, errors.New("failed to decode StaticExtra")
	}

	hasTradeEvent := false
	hasParametersSetEvent := false
	hasOracleEvent := false

	// Process events if provided
	for _, log := range params.Logs {
		if !isLogFromPool(log, p.Address) {
			// Check if it's from one of the oracle contracts
			if log.Address == staticExtra.Oracles[0] || log.Address == staticExtra.Oracles[1] {
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
		return t.handleParametersSetEvent(ctx, p, params)
	}

	// If we have oracle events, decode and update oracle rates
	// Oracle rate changes don't affect reserves but do affect pricing
	if hasOracleEvent {
		updatedPool, err := t.handleOracleEvents(ctx, p, params, staticExtra)
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
		updatedExtra, err := t.fetchPoolReservesFromNode(ctx, p, staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": p.Address,
				"error":       err,
			}).Error("Failed to fetch pool reserves after trade event")
			return p, err
		}

		updateReserves(p, updatedExtra)

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
	var updatedExtra Extra
	var err error

	// Fetch everything including oracle rates
	updatedExtra, err = t.fetchPoolStateWithOraclesFromNode(ctx, p, staticExtra)

	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Error("Failed to fetch pool state")
		return p, err
	}

	updateReserves(p, updatedExtra)

	// Update extra data
	extraBytes, err := json.Marshal(updatedExtra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.SwapFee = updatedExtra.Epsilon.Float64() / 1e18
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("Finished getting new state of pool")

	return p, nil
}

func updateReserves(p entity.Pool, updatedExtra Extra) {
	for i, reserve := range updatedExtra.Reserves {
		reserve = mulu(reserve, big256.TenPow(p.Tokens[i].Decimals))
		reserve.MulDivOverflow(reserve, OracleDecimals, updatedExtra.OracleRates[i])
		p.Reserves[i] = reserve.String()
	}
}

// handleOracleEvents processes Chainlink oracle events (AnswerUpdated or NewTransmission)
// and updates the oracle rates in the pool state
func (t *PoolTracker) handleOracleEvents(_ context.Context, p entity.Pool, params pool.GetNewPoolStateParams,
	staticExtra StaticExtra) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Info("Processing oracle events")

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Warn("Failed to decode existing extra data")
	}
	hasBaseOracleUpdate := false
	hasQuoteOracleUpdate := false

	// Process oracle events
	for _, log := range params.Logs {
		// Check if it's from base oracle
		if log.Address == staticExtra.Oracles[0] {
			if isAnswerUpdatedEvent(log) {
				// AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt)
				// Topics: [0] = event sig, [1] = current (indexed), [2] = roundId (indexed)
				if len(log.Topics) >= 2 {
					// Extract current price from indexed topic
					extra.OracleRates[0] = new(uint256.Int).SetBytes(log.Topics[1].Bytes())
					hasBaseOracleUpdate = true
					logger.WithFields(logger.Fields{
						"dex":         DexType,
						"poolAddress": p.Address,
						"oracle":      "base",
						"rate":        extra.OracleRates[0],
					}).Debug("Updated base oracle rate from AnswerUpdated event")
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
				extra.OracleRates[0] = uint256.MustFromBig(event.Answer)
				hasBaseOracleUpdate = true
				logger.WithFields(logger.Fields{
					"dex":         DexType,
					"poolAddress": p.Address,
					"oracle":      "base",
					"rate":        event.Answer,
				}).Info("Updated base oracle rate from NewTransmission event")
			}
		}

		// Check if it's from quote oracle
		if log.Address == staticExtra.Oracles[1] {
			if isAnswerUpdatedEvent(log) {
				// AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt)
				if len(log.Topics) >= 2 {
					extra.OracleRates[1] = new(uint256.Int).SetBytes(log.Topics[1].Bytes())
					hasQuoteOracleUpdate = true
					logger.WithFields(logger.Fields{
						"dex":         DexType,
						"poolAddress": p.Address,
						"oracle":      "quote",
						"rate":        extra.OracleRates[1],
					}).Debug("Updated quote oracle rate from AnswerUpdated event")
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
				extra.OracleRates[1] = uint256.MustFromBig(event.Answer)
				hasQuoteOracleUpdate = true
				logger.WithFields(logger.Fields{
					"dex":         DexType,
					"poolAddress": p.Address,
					"oracle":      "quote",
					"rate":        event.Answer,
				}).Info("Updated quote oracle rate from NewTransmission event")
			}
		}
	}

	if !hasBaseOracleUpdate && !hasQuoteOracleUpdate {
		return p, fmt.Errorf("no oracle events decoded")
	}

	// Recalculate derived oracle rate if both rates are available
	if extra.OracleRates[0] != nil && extra.OracleRates[1] != nil {
		// oracleRate = baseRate / quoteRate (scaled by precision)
		extra.OracleRate, _ = new(uint256.Int).MulDivOverflow(extra.OracleRates[0], big256.BONE, extra.OracleRates[1])
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
		"baseRate":    extra.OracleRates[0],
		"quoteRate":   extra.OracleRates[1],
		"derivedRate": extra.OracleRate,
	}).Debug("Successfully updated oracle rates from events")

	return p, nil
}

// fetchOracleRates fetches the latest prices from Chainlink oracles with fallback
// Tries latestAnswer() first, then falls back to latestRoundData() for newer aggregators
func (t *PoolTracker) fetchOracleRates(ctx context.Context, oracles [2]common.Address) ([2]*big.Int, error) {
	// Try latestAnswer() first (simpler, older aggregators)
	var rates [2]*big.Int
	oracleAddr0, oracleAddr1 := hexutil.Encode(oracles[0][:]), hexutil.Encode(oracles[1][:])
	_, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    chainlinkAggregatorABI,
		Target: oracleAddr0,
		Method: oracleMethodLatestAnswer,
	}, []any{&rates[0]}).AddCall(&ethrpc.Call{
		ABI:    chainlinkAggregatorABI,
		Target: oracleAddr1,
		Method: oracleMethodLatestAnswer,
	}, []any{&rates[1]}).TryAggregate()
	rate0Ok, rate1Ok := rates[0] != nil && rates[0].Sign() > 0, rates[1] != nil && rates[1].Sign() > 0
	if err == nil && rate0Ok && rate1Ok {
		return rates, nil
	}

	// Fallback to latestRoundData() for newer aggregators
	// latestRoundData() returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
	var roundData [2]struct {
		RoundId         *big.Int
		Answer          *big.Int
		StartedAt       *big.Int
		UpdatedAt       *big.Int
		AnsweredInRound *big.Int
	}
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if !rate0Ok {
		req.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: oracleAddr0,
			Method: oracleMethodLatestRoundData,
		}, []any{&roundData[0]})
	}
	if !rate1Ok {
		req.AddCall(&ethrpc.Call{
			ABI:    chainlinkAggregatorABI,
			Target: oracleAddr1,
			Method: oracleMethodLatestRoundData,
		}, []any{&roundData[1]})
	}
	if _, err := req.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dex":     DexType,
			"oracles": oracles,
			"error":   err,
		}).Warn("Both latestAnswer() and latestRoundData() failed for oracle")
		return [2]*big.Int{}, err
	}

	if !rate0Ok {
		if roundData[0].Answer == nil || roundData[0].Answer.Sign() <= 0 {
			return [2]*big.Int{}, fmt.Errorf("invalid oracle answer: %v", roundData[0].Answer)
		}
		rates[0] = roundData[0].Answer
	}
	if !rate1Ok {
		if roundData[1].Answer == nil || roundData[1].Answer.Sign() <= 1 {
			return [2]*big.Int{}, fmt.Errorf("invalid oracle answer: %v", roundData[1].Answer)
		}
		rates[1] = roundData[1].Answer
	}

	return rates, nil
}

// fetchPoolStateFromNode fetches current reserves, curve parameters, and oracle rates from the blockchain
func (t *PoolTracker) fetchPoolStateFromNode(ctx context.Context, poolAddress string) (Extra, error) {
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

	rpcRequest := t.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
	}, []any{&liquidityResult})

	// Fetch curve parameters using viewCurve() method
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
	}, []any{&curveResult})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return Extra{}, err
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParams{
		Alpha:   uint256.MustFromBig(curveResult.Alpha),
		Beta:    uint256.MustFromBig(curveResult.Beta),
		Delta:   uint256.MustFromBig(curveResult.Delta),
		Epsilon: uint256.MustFromBig(curveResult.Epsilon),
		Lambda:  uint256.MustFromBig(curveResult.Lambda),
	}

	// Build extra without oracle data (will be populated by fetchOracleRates)
	extra := Extra{
		CurveParams: curveParams,
		Reserves:    [2]*uint256.Int{uint256.MustFromBig(reserves[0]), uint256.MustFromBig(reserves[1])},
	}

	return extra, nil
}

// fetchPoolStateWithOraclesFromNode fetches reserves, curve parameters, AND oracle rates from the blockchain
func (t *PoolTracker) fetchPoolStateWithOraclesFromNode(ctx context.Context, p entity.Pool,
	staticExtra StaticExtra) (Extra, error) {
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

	rpcRequest := t.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodLiquidity,
	}, []any{&liquidityResult})
	// Fetch curve parameters using viewCurve() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodViewCurve,
	}, []any{&curveResult})
	if _, err := rpcRequest.Aggregate(); err != nil {
		return Extra{}, fmt.Errorf("failed to fetch pool state with oracles: %w", err)
	}

	oracleRates, err := t.fetchOracleRates(ctx, staticExtra.Oracles)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"oracles":     staticExtra.Oracles,
			"error":       err,
		}).Warn("Failed to fetch oracle rates")
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParams{
		Alpha:   uint256.MustFromBig(curveResult.Alpha),
		Beta:    uint256.MustFromBig(curveResult.Beta),
		Delta:   uint256.MustFromBig(curveResult.Delta),
		Epsilon: uint256.MustFromBig(curveResult.Epsilon),
		Lambda:  uint256.MustFromBig(curveResult.Lambda),
	}

	// Build extra with oracle data
	extra := Extra{
		CurveParams: curveParams,
		Reserves:    [2]*uint256.Int{uint256.MustFromBig(reserves[0]), uint256.MustFromBig(reserves[1])},
		OracleRates: [2]*uint256.Int{uint256.MustFromBig(oracleRates[0]), uint256.MustFromBig(oracleRates[1])},
	}

	// oracleRate = baseRate / quoteRate (scaled by precision)
	extra.OracleRate, _ = new(uint256.Int).MulDivOverflow(extra.OracleRates[0], big256.BONE, extra.OracleRates[1])

	return extra, nil
}

// fetchPoolReservesFromNode fetches only reserves (and optionally oracle rates) - efficient for Trade events
func (t *PoolTracker) fetchPoolReservesFromNode(ctx context.Context, p entity.Pool,
	staticExtra StaticExtra) (Extra, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"error":       err,
		}).Warn("Failed to decode existing extra data")
	}
	// Define struct to match the liquidity() return signature
	type LiquidityResult struct {
		Total      *big.Int
		Individual []*big.Int
	}

	var liquidityResult LiquidityResult

	rpcRequest := t.ethrpcClient.NewRequest().SetContext(ctx)
	// Fetch reserves using liquidity() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodLiquidity,
	}, []any{&liquidityResult})
	if _, err := rpcRequest.Aggregate(); err != nil {
		return Extra{}, fmt.Errorf("failed to fetch pool state with oracles: %w", err)
	}

	// Also fetch oracle rates if addresses are available (good to refresh periodically)
	oracleRates, err := t.fetchOracleRates(ctx, staticExtra.Oracles)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":         DexType,
			"poolAddress": p.Address,
			"oracles":     staticExtra.Oracles,
			"error":       err,
		}).Warn("Failed to fetch oracle rates")
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return Extra{}, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	extra.Reserves = [2]*uint256.Int{uint256.MustFromBig(reserves[0]), uint256.MustFromBig(reserves[1])}
	extra.OracleRates = [2]*uint256.Int{uint256.MustFromBig(oracleRates[0]), uint256.MustFromBig(oracleRates[1])}
	extra.OracleRate, _ = new(uint256.Int).MulDivOverflow(extra.OracleRates[0], big256.BONE, extra.OracleRates[1])

	// Return existing extra with updated oracle rates (curve parameters unchanged)
	return extra, nil
}

// handleParametersSetEvent processes a ParametersSet event by decoding the event data
// ParametersSet(uint256 alpha, uint256 beta, uint256 delta, uint256 epsilon, uint256 lambda)
// This is emitted when admin updates curve parameters - infrequent but critical for pricing
func (t *PoolTracker) handleParametersSetEvent(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams) (entity.Pool, error) {
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
		extra.CurveParams = CurveParams{
			Alpha:   uint256.MustFromBig(event.Alpha),
			Beta:    uint256.MustFromBig(event.Beta),
			Delta:   uint256.MustFromBig(event.Delta),
			Epsilon: uint256.MustFromBig(event.Epsilon),
			Lambda:  uint256.MustFromBig(event.Lambda),
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
		p.SwapFee = extra.Epsilon.Float64() / 1e18
		p.Timestamp = time.Now().Unix()

		return p, nil
	}

	// If we couldn't find/decode the event, fall back to RPC call
	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"poolAddress": p.Address,
	}).Warn("No ParametersSet event found in logs, falling back to RPC")

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

	rpcRequest := t.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodLiquidity,
	}, []any{&liquidityResult})

	// Fetch curve parameters using viewCurve() method
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: p.Address,
		Method: poolMethodViewCurve,
	}, []any{&curveResult})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return p, err
	}

	// Convert reserves
	reserves := liquidityResult.Individual
	if len(reserves) != 2 {
		return p, fmt.Errorf("expected 2 reserves, got %d", len(reserves))
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParams{
		Alpha:   uint256.MustFromBig(curveResult.Alpha),
		Beta:    uint256.MustFromBig(curveResult.Beta),
		Delta:   uint256.MustFromBig(curveResult.Delta),
		Epsilon: uint256.MustFromBig(curveResult.Epsilon),
		Lambda:  uint256.MustFromBig(curveResult.Lambda),
	}

	extra.CurveParams = curveParams
	extra.Reserves = [2]*uint256.Int{uint256.MustFromBig(reserves[0]), uint256.MustFromBig(reserves[1])}
	updateReserves(p, extra)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.SwapFee = extra.Epsilon.Float64() / 1e18
	p.Timestamp = time.Now().Unix()

	return p, nil
}
