package slyngfun

import (
	"context"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState reads the whole curve at one block. It runs with or without logs: the opening
// surcharge falls away with time alone, and a graduation can be triggered by anyone calling
// graduate() rather than by a trade this source would see.
//
// The launchpad's trade constants never change, but they are read every pass with the rest:
// discovery is a pure log decode, so the first tracker pass is the first point they are known.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"dexId": t.config.DexID, "address": p.Address})

	var (
		curve       curveResp
		tradeFeeBps *big.Int
		snipeBps    *big.Int
		snipeWindow *big.Int
	)

	token := common.HexToAddress(p.Address)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: t.config.Launchpad, Method: launchpadMethodCurves,
		Params: []any{token}}, []any{&curve})
	for _, c := range []struct {
		method string
		out    any
	}{
		{launchpadMethodTradeFeeBps, &tradeFeeBps},
		{launchpadMethodSnipeBps, &snipeBps},
		{launchpadMethodSnipeWindow, &snipeWindow},
	} {
		req.AddCall(&ethrpc.Call{ABI: launchpadABI, Target: t.config.Launchpad, Method: c.method}, []any{c.out})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to fetch curve state")
		return p, err
	}
	if slices.Contains(resp.Result, false) || resp.BlockNumber == nil {
		return p, ErrFailedCall
	}
	if curve.QuoteReserve == nil || curve.TokenReserve == nil || curve.GraduationTarget == nil ||
		curve.VirtualQuote == nil || tradeFeeBps == nil || snipeBps == nil || snipeWindow == nil {
		return p, ErrFailedCall
	}
	if !curve.Exists {
		return p, ErrUnknownCurve
	}

	extra := Extra{
		QuoteReserve: uint256.MustFromBig(curve.QuoteReserve),
		TokenReserve: uint256.MustFromBig(curve.TokenReserve),
		Graduated:    curve.Graduated,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	staticExtra := StaticExtra{
		Launchpad:          strings.ToLower(t.config.Launchpad),
		IsNativeQuote:      valueobject.IsNativeOrZeroAddr(curve.Quote),
		GraduationTarget:   uint256.MustFromBig(curve.GraduationTarget),
		VirtualQuote:       uint256.MustFromBig(curve.VirtualQuote),
		CreatedAt:          curve.CreatedAt,
		TradeFeeBps:        tradeFeeBps.Uint64(),
		SnipeBps:           snipeBps.Uint64(),
		SnipeWindowSeconds: snipeWindow.Uint64(),
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.StaticExtra = string(staticExtraBytes)
	p.SwapFee = float64(staticExtra.TradeFeeBps)
	// What each side can still pay out: the quote the curve holds, and the tokens it has left.
	p.Reserves = entity.PoolReserves{extra.QuoteReserve.Dec(), extra.TokenReserve.Dec()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	if curve.Graduated {
		// A graduated curve is done for good: its liquidity is in a Uniswap v4 pool the
		// uniswap-v4 source prices. Timestamp 1 (not 0, which pool-service treats as
		// always-active) makes it look maximally stale so it is archived.
		p.Timestamp = 1
	} else {
		p.Timestamp = time.Now().Unix()
	}

	return p, nil
}
