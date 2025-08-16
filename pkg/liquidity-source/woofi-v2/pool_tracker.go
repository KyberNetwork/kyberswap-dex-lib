package woofiv2

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeWooFiV2, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.WithFields(logger.Fields{
			"pool_id":      p.Address,
			"duration_ms:": time.Since(startTime).Milliseconds(),
		})
	}()

	type WoStateContractType struct {
		Price      *big.Int `json:"price"`
		Spread     uint64   `json:"spread"`
		Coeff      uint64   `json:"coeff"`
		WoFeasible bool     `json:"woFeasible"`
	}

	type clOracleResp struct {
		Oracle       common.Address `json:"oracle"`
		Decimal      uint8          `json:"decimal"`
		CloPreferred bool           `json:"cloPreferred"`
	}

	var (
		isPaused                 bool
		quoteToken, wooracle     common.Address
		timestamp, staleDuration *big.Int
		bound                    uint64
		priceTokenDecimals       = make([]uint8, len(p.Tokens))
		tokenInfos               = make([]struct {
			Reserve *big.Int `json:"reserve"`
			FeeRate uint16   `json:"feeRate"`
		}, len(p.Tokens))
		woState   = make([]struct{ WoStateContractType }, len(p.Tokens))
		clOracles = make([]clOracleResp, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodPaused,
		Params: nil,
	}, []any{&isPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodQuoteToken,
		Params: nil,
	}, []any{&quoteToken})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodWooracle,
		Params: nil,
	}, []any{&wooracle})
	for i, token := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    WooPPV2ABI,
			Target: p.Address,
			Method: wooPPV2MethodTokenInfos,
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&tokenInfos[i]})
	}

	callsResult, err := calls.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("[WooFiV2] failed to aggregate call")
		return entity.Pool{}, err
	}

	if isPaused {
		extraBytes, err := json.Marshal(&Extra{
			IsPaused: true,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"err":         err,
			}).Errorf("failed to marshal extra data")
			return entity.Pool{}, err
		}

		p.Extra = string(extraBytes)
		p.Reserves = lo.Map(p.Reserves, func(_ string, _ int) string { return "0" })
	}

	blockNumber := callsResult.BlockNumber

	oracleCalls := d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodTimestamp,
		Params: nil,
	}, []any{&timestamp})
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodStaleDuration,
		Params: nil,
	}, []any{&staleDuration})
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodBound,
		Params: nil,
	}, []any{&bound})
	for i, token := range p.Tokens {
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodWoState,
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&woState[i]})
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodDecimals,
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&priceTokenDecimals[i]})
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodClOracles,
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&clOracles[i]})
	}
	if _, err := oracleCalls.TryBlockAndAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("[WooFiV2] failed to aggregate call")
		return entity.Pool{}, err
	}

	// Call ChainLink Oracle to get lastestRoundData
	latestRoundData := make([]struct {
		RoundId         *big.Int `json:"roundId" abi:"roundId"`
		Answer          *big.Int `json:"answer" abi:"answer"`
		StartedAt       *big.Int `json:"startedAt" abi:"startedAt"`
		UpdatedAt       *big.Int `json:"updatedAt" abi:"updatedAt"`
		AnsweredInRound *big.Int `json:"answeredInRound" abi:"answeredInRound"`
	}, len(p.Tokens))

	if _, ok := lo.Find(clOracles, func(clOracle clOracleResp) bool {
		return clOracle.Oracle.Cmp(zeroAddress) == 0
	}); !ok {
		cloracleCalls := d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
		for i := range p.Tokens {
			cloracleCalls.AddCall(&ethrpc.Call{
				ABI:    CloracleABI,
				Target: clOracles[i].Oracle.Hex(),
				Method: cloracleMethodLatestRoundData,
				Params: nil,
			}, []any{&latestRoundData[i]})
		}
		if _, err := cloracleCalls.TryBlockAndAggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"err":         err,
			}).Errorf("[WooFiV2] failed to aggregate call to chainlink oracle")
			return entity.Pool{}, err
		}
	}

	poolCloracle := make(map[string]Cloracle, len(p.Tokens))
	for i, token := range p.Tokens {
		answer, _ := uint256.FromBig(latestRoundData[i].Answer)
		updatedAt, _ := uint256.FromBig(latestRoundData[i].UpdatedAt)

		poolCloracle[token.Address] = Cloracle{
			OracleAddress: clOracles[i].Oracle,
			Answer:        answer,
			UpdatedAt:     updatedAt,
			CloPreferred:  clOracles[i].CloPreferred,
		}
	}

	extraTokenInfos := make(map[string]TokenInfo)
	extraStates := make(map[string]State)
	extraDecimals := make(map[string]uint8)
	reserves := make(entity.PoolReserves, len(p.Tokens))

	for i, token := range p.Tokens {
		tokenInfoReserve, overflow := uint256.FromBig(tokenInfos[i].Reserve)
		if overflow {
			return entity.Pool{}, errors.New("reserve overflow")
		}

		price, overflow := uint256.FromBig(woState[i].Price)
		if overflow {
			return entity.Pool{}, errors.New("price overflow")
		}

		extraTokenInfos[token.Address] = TokenInfo{
			Reserve: tokenInfoReserve,
			FeeRate: tokenInfos[i].FeeRate,
		}
		extraStates[token.Address] = State{
			Price:      price,
			Spread:     woState[i].Spread,
			Coeff:      woState[i].Coeff,
			WoFeasible: woState[i].WoFeasible,
		}
		extraDecimals[token.Address] = priceTokenDecimals[i]
		reserves[i] = tokenInfos[i].Reserve.String()
	}

	extraBytes, err := json.Marshal(&Extra{
		QuoteToken: strings.ToLower(quoteToken.Hex()),
		TokenInfos: extraTokenInfos,
		Wooracle: Wooracle{
			Address:       wooracle.Hex(),
			States:        extraStates,
			Decimals:      extraDecimals,
			Timestamp:     timestamp.Int64(),
			StaleDuration: staleDuration.Int64(),
			Bound:         bound,
		},
		Cloracle: poolCloracle,
		IsPaused: false,
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	logger.WithFields(logger.Fields{
		"address": p.Address,
		"type":    p.Type,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
