package lunyafun

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState reads the whole curve at one block. It runs with or without logs: the anti-snipe
// surcharge decays with time alone, and a launch that has graduated stops trading without this source
// seeing anything of its own.
//
// The terms below Sold never change, but they are read every pass all the same: discovery is a pure log
// decode, so the first tracker pass is the first point they are known.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"dexId": t.config.DexID, "address": p.Address})

	var (
		phase    uint8
		reserve  *big.Int
		sold     *big.Int
		openedAt *big.Int
		cfg      launchConfigResp
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	for _, c := range []struct {
		method string
		out    any
	}{
		{launchMethodPhase, &phase},
		{launchMethodReserve, &reserve},
		{launchMethodSold, &sold},
		{launchMethodOpenedAt, &openedAt},
		{launchMethodConfig, &cfg},
	} {
		req.AddCall(&ethrpc.Call{ABI: launchABI, Target: p.Address, Method: c.method}, []any{c.out})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to fetch launch state")
		return p, err
	}
	if slices.Contains(resp.Result, false) || resp.BlockNumber == nil {
		return p, ErrFailedCall
	}

	extra := Extra{
		Phase:        phase,
		Reserve:      uint256.MustFromBig(reserve),
		Sold:         uint256.MustFromBig(sold),
		VirtualQuote: uint256.MustFromBig(cfg.VirtualQuote),
		VirtualToken: uint256.MustFromBig(cfg.VirtualToken),
		CurveSupply:  uint256.MustFromBig(cfg.CurveSupply),
		CurveFeeBps:  cfg.CurveFeeBps,
		SnipeTaxBps:  cfg.SnipeTaxBps,
		SnipeWindow:  cfg.SnipeWindow,
		SnipeDecay:   cfg.SnipeDecay,
		OpenedAt:     openedAt.Uint64(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// What each side can still pay out: the quote the curve holds, and the tokens it has left to sell.
	tokensLeft := new(big.Int)
	if cfg.CurveSupply.Cmp(sold) > 0 {
		tokensLeft.Sub(cfg.CurveSupply, sold)
	}

	p.Extra = string(extraBytes)
	p.SwapFee = float64(cfg.CurveFeeBps)
	p.Reserves = entity.PoolReserves{reserve.String(), tokensLeft.String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()

	return p, nil
}
