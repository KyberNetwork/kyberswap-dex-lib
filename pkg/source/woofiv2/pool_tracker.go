package woofiv2

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new states of pool", p.Type)

	type WoStateContractType struct {
		Price      *big.Int `json:"price"`
		Spread     uint64   `json:"spread"`
		Coeff      uint64   `json:"coeff"`
		WoFeasible bool     `json:"woFeasible"`
	}

	var (
		quoteToken, wooracle                   common.Address
		unclaimedFee, timestamp, staleDuration *big.Int
		bound                                  uint64
		tokenDecimals                          = make([]uint8, len(p.Tokens))
		priceTokenDecimals                     = make([]uint8, len(p.Tokens))
		tokenInfos                             = make([]struct {
			Reserve *big.Int `json:"reserve"`
			FeeRate uint16   `json:"feeRate"`
		}, len(p.Tokens))
		woState = make([]struct{ WoStateContractType }, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodQuoteToken,
		Params: nil,
	}, []interface{}{&quoteToken})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodUnclaimedFee,
		Params: nil,
	}, []interface{}{&unclaimedFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    WooPPV2ABI,
		Target: p.Address,
		Method: wooPPV2MethodWooracle,
		Params: nil,
	}, []interface{}{&wooracle})
	for i, token := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    WooPPV2ABI,
			Target: p.Address,
			Method: wooPPV2MethodTokenInfos,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&tokenInfos[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    Erc20ABI,
			Target: token.Address,
			Method: erc20MethodDecimals,
			Params: nil,
		}, []interface{}{&tokenDecimals[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("[WooFiV2] failed to aggregate call")
		return entity.Pool{}, err
	}

	type CloPriceResp struct {
		RefPrice     *big.Int
		RefTimestamp *big.Int
	}
	type CloOraclesResp struct {
		Oracle       common.Address
		Decimal      uint8
		CloPreferred bool
	}
	var (
		cloPriceResp  = make([]CloPriceResp, len(p.Tokens))
		cloOracleResp = make([]CloOraclesResp, len(p.Tokens))
	)
	oracleCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodTimestamp,
		Params: nil,
	}, []interface{}{&timestamp})
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodStaleDuration,
		Params: nil,
	}, []interface{}{&staleDuration})
	oracleCalls.AddCall(&ethrpc.Call{
		ABI:    WooracleV2ABI,
		Target: wooracle.Hex(),
		Method: wooracleMethodBound,
		Params: nil,
	}, []interface{}{&bound})
	for i, token := range p.Tokens {
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodWoState,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&woState[i]})
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodDecimals,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&priceTokenDecimals[i]})
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodClOracles,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&cloOracleResp[i]})
		oracleCalls.AddCall(&ethrpc.Call{
			ABI:    WooracleV2ABI,
			Target: wooracle.Hex(),
			Method: wooracleMethodCloPrice,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&cloPriceResp[i]})
	}
	if _, err := oracleCalls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("[WooFiV2] failed to aggregate call")
		return entity.Pool{}, err
	}

	extraTokenInfo := make(map[string]*TokenInfo)
	reserves := make(entity.PoolReserves, len(p.Tokens))

	for i, token := range p.Tokens {
		extraTokenInfo[token.Address] = &TokenInfo{
			Reserve:  tokenInfos[i].Reserve,
			FeeRate:  big.NewInt(int64(tokenInfos[i].FeeRate)),
			Decimals: tokenDecimals[i],
			State: &OracleState{
				Price:        woState[i].Price,
				Spread:       big.NewInt(int64(woState[i].Spread)),
				Coeff:        big.NewInt(int64(woState[i].Coeff)),
				WoFeasible:   woState[i].WoFeasible,
				Decimals:     priceTokenDecimals[i],
				CloPrice:     cloPriceResp[i].RefPrice,
				CloPreferred: cloOracleResp[i].CloPreferred,
			},
		}
		reserves[i] = tokenInfos[i].Reserve.String()
	}

	extraBytes, err := json.Marshal(&Extra{
		QuoteToken:    strings.ToLower(quoteToken.Hex()),
		UnclaimedFee:  unclaimedFee,
		Wooracle:      wooracle.Hex(),
		TokenInfos:    extraTokenInfo,
		Timestamp:     timestamp,
		StaleDuration: staleDuration,
		Bound:         big.NewInt(int64(bound)),
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

	logger.WithFields(logger.Fields{
		"address": p.Address,
		"type":    p.Type,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
