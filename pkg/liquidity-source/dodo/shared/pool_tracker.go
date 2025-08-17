package shared

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	cmap "github.com/orcaman/concurrent-map"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	blackList    cmap.ConcurrentMap
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	blackList, err := InitBlackList(cfg.BlacklistFilePath)
	if err != nil {
		return nil, err
	}

	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		blackList:    blackList,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtraData = struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtraData); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to unmarshal extra data")
		return entity.Pool{}, err
	}
	if staticExtraData.Type == SubgraphPoolTypeDodoClassical {
		return d.getNewPoolStateDodoV1(ctx, p)
	}

	return d.getNewPoolStateDodoV2(ctx, p)
}

func (d *PoolTracker) getNewPoolStateDodoV1(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[%v] Start getting new state of pool: %v", p.Type, p.Address)

	var (
		targetReserve                                         V1TargetReserve
		i, k, lpFeeRate, mtFeeRate, baseReserve, quoteReserve *big.Int
		rStatus                                               uint8
		tradeAllowed, sellingAllowed, buyingAllowed           bool
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodGetExpectedTarget,
		Params: nil,
	}, []any{&targetReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodK,
		Params: nil,
	}, []any{&k})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodRStatus,
		Params: nil,
	}, []any{&rStatus})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodGetOraclePrice,
		Params: nil,
	}, []any{&i})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodLpFeeRate,
		Params: nil,
	}, []any{&lpFeeRate})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodMtFeeRate,
		Params: nil,
	}, []any{&mtFeeRate})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodBaseBalance,
		Params: nil,
	}, []any{&baseReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodQuoteBalance,
		Params: nil,
	}, []any{&quoteReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    v1PoolABI,
		Target: p.Address,
		Method: dodoV1MethodTradeAllowed,
		Params: nil,
	}, []any{&tradeAllowed})

	if d.config.ChainID != valueobject.ChainIDEthereum && p.Type != string(valueobject.ExchangeDodoClassical) {
		calls.AddCall(&ethrpc.Call{
			ABI:    v1PoolABI,
			Target: p.Address,
			Method: dodoV1MethodSellingAllowed,
			Params: nil,
		}, []any{&sellingAllowed})

		calls.AddCall(&ethrpc.Call{
			ABI:    v1PoolABI,
			Target: p.Address,
			Method: dodoV1MethodBuyingAllowed,
			Params: nil,
		}, []any{&buyingAllowed})
	} else {
		sellingAllowed = true
		buyingAllowed = true
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[DodoV1] failed to aggregate for pool data")

		return entity.Pool{}, err
	}

	extra := V1Extra{
		B:              number.SetFromBig(baseReserve),
		Q:              number.SetFromBig(quoteReserve),
		B0:             number.SetFromBig(targetReserve.BaseTarget),
		Q0:             number.SetFromBig(targetReserve.QuoteTarget),
		RStatus:        int(rStatus),
		OraclePrice:    number.SetFromBig(i),
		K:              number.SetFromBig(k),
		MtFeeRate:      number.SetFromBig(mtFeeRate),
		LpFeeRate:      number.SetFromBig(lpFeeRate),
		TradeAllowed:   tradeAllowed,
		SellingAllowed: sellingAllowed,
		BuyingAllowed:  buyingAllowed,
		Swappable:      tradeAllowed && sellingAllowed,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshaling the extra bytes data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = extra.LpFeeRate.Float64() + extra.MtFeeRate.Float64()
	p.Reserves = entity.PoolReserves{baseReserve.String(), quoteReserve.String()}
	p.Timestamp = time.Now().Unix()

	logger.Infof("[%v] Finish getting new state of pool: %v", p.Type, p.Address)

	return p, nil
}

func (d *PoolTracker) getNewPoolStateDodoV2(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[%v] Start getting new state of pool: %v", p.Type, p.Address)

	_, ok := d.blackList.Get(p.Address)
	if ok {
		return entity.Pool{}, ErrPoolAddressBanned
	}

	var (
		state     V2PMMState
		feeRate   V2FeeRate
		lpFeeRate *big.Int
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    V2PoolABI,
		Target: p.Address,
		Method: dodoV2MethodGetPMMStateForCall,
		Params: nil,
	}, []any{&state})
	calls.AddCall(&ethrpc.Call{
		ABI:    V2PoolABI,
		Target: p.Address,
		Method: dodoV1MethodLpFeeRate,
		Params: nil,
	}, []any{&lpFeeRate})
	calls.AddCall(&ethrpc.Call{
		ABI:    V2PoolABI,
		Target: p.Address,
		Method: dodoV2MethodGetUserFeeRate,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&feeRate})

	if _, err := calls.TryBlockAndAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[DodoV2] failed to aggregate for pool data")
		if p.Extra != "" {
			p.Extra = ""
			p.Reserves = entity.PoolReserves{"0", "0"}
			return p, nil
		}
		return entity.Pool{}, err
	}

	if state.B == nil && state.Q == nil &&
		state.B0 == nil && state.Q0 == nil &&
		state.I == nil && state.K == nil && state.R == nil {

		return entity.Pool{}, fmt.Errorf("get pool state failed")
	}

	if feeRate.MtFeeRate == nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"exchange":    p.Exchange,
		}).Errorf("[DodoV2] added pool to blacklist")

		if state.K.Sign() == 0 && (state.B.Sign() == 0 || state.Q.Sign() == 0) {
			d.blackList.Set(p.Address, struct{}{})
		}

		return entity.Pool{}, fmt.Errorf("get pool feeRate failed")
	}

	extra := V2Extra{
		I:         number.SetFromBig(state.I),
		K:         number.SetFromBig(state.K),
		B:         number.SetFromBig(state.B),
		Q:         number.SetFromBig(state.Q),
		B0:        number.SetFromBig(state.B0),
		Q0:        number.SetFromBig(state.Q0),
		R:         number.SetFromBig(state.R),
		MtFeeRate: number.SetFromBig(feeRate.MtFeeRate),
		LpFeeRate: number.SetFromBig(lpFeeRate),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshaling the extra bytes data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = float64(lpFeeRate.Int64() + feeRate.MtFeeRate.Int64())
	p.Reserves = entity.PoolReserves{state.B.String(), state.Q.String()}
	p.Timestamp = time.Now().Unix()

	logger.Infof("[%v] Finish updating state of pool: %v", p.Type, p.Address)

	return p, nil
}
