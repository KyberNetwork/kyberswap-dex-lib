package wasabiprop

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	samples := make([][][2]*big.Int, 2)
	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		start := lo.Ternary(p.Tokens[i].Decimals < sampleSize/2, 0, p.Tokens[i].Decimals-sampleSize/2)
		idx := 0
		for k := start; k <= start+sampleSize-1 && idx < sampleSize; k++ {
			samples[i][idx] = [2]*big.Int{bignumber.TenPowInt(k), big.NewInt(0)}
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: "quoteExactInput",
				Params: []any{
					common.HexToAddress(p.Tokens[i].Address),
					samples[i][idx][0],
				},
			}, []any{&samples[i][idx][1]})
			idx++
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	// Warn on quote gaps: >0 ... 0 ... >0 as amountIn increases.
	t.warnGapInQuotes(p, samples)

	if t.cfg.Buffer > 0 {
		buf := big.NewInt(t.cfg.Buffer)
		for i := range samples {
			for j := range samples[i] {
				if samples[i][j][1] != nil {
					samples[i][j][1].Mul(samples[i][j][1], buf)
					samples[i][j][1].Div(samples[i][j][1], bignumber.BasisPoint)
				}
			}
		}
	}

	for i := range samples {
		valid := samples[i][:0]
		for _, s := range samples[i] {
			if s[0] != nil && s[1] != nil {
				valid = append(valid, s)
			}
		}
		samples[i] = valid
	}

	// Get reserves from pool (returns a struct/tuple)
	var reserves getReservesResult
	reqRes := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	reqRes.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "getReserves",
	}, []any{&reserves})

	if _, err := reqRes.Call(); err != nil {
		return p, err
	}

	if reserves.BaseTokenReserves == nil || reserves.QuoteTokenReserves == nil {
		return p, ErrInsufficientLiquidity
	}

	p.Reserves = []string{
		reserves.BaseTokenReserves.String(),
		reserves.QuoteTokenReserves.String(),
	}

	for dir := range samples {
		valid := samples[dir][:0]
		for _, s := range samples[dir] {
			if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
				continue
			}
			valid = append(valid, s)
		}
		samples[dir] = valid
	}

	extra := Extra{Samples: samples}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	p.Timestamp = time.Now().Unix()
	p.BlockNumber = res.BlockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) warnGapInQuotes(p entity.Pool, samples [][][2]*big.Int) {
	for dir := range samples {
		seenPositive := false
		zeroRunStart := -1

		for i := range samples[dir] {
			pt := samples[dir][i]
			if pt[0] == nil || pt[1] == nil {
				continue
			}
			if pt[1].Sign() > 0 {
				if zeroRunStart >= 0 {
					startAmt := samples[dir][zeroRunStart][0]
					endAmt := samples[dir][i-1][0]
					logger.WithFields(logger.Fields{
						"pool":           p.Address,
						"dir":            dir,
						"tokenIn":        p.Tokens[dir].Address,
						"tokenOut":       p.Tokens[1-dir].Address,
						"holeFromAmount": startAmt.String(),
						"holeToAmount":   endAmt.String(),
						"resumeAmount":   pt[0].String(),
					}).Warn("wasabi-prop quote gap detected (positive -> zero -> positive)")
					zeroRunStart = -1
				}
				seenPositive = true
				continue
			}
			if seenPositive && zeroRunStart < 0 {
				zeroRunStart = i
			}
		}
	}
}
