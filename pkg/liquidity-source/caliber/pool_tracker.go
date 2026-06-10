package caliber

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: client}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	pairID := common.HexToHash(staticExtra.PairID)
	token0 := common.HexToAddress(p.Tokens[0].Address)
	token1 := common.HexToAddress(p.Tokens[1].Address)

	var balances struct {
		ReserveX *big.Int
		ReserveY *big.Int
	}
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    caliberABI,
		Target: staticExtra.Contract,
		Method: methodGetPoolBalances,
		Params: []any{pairID},
	}, []any{&balances})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber

	r0 := uint256.MustFromBig(balances.ReserveX)
	r1 := uint256.MustFromBig(balances.ReserveY)

	points0 := buildPoints(balances.ReserveX)
	points1 := buildPoints(balances.ReserveY)
	ladder0, ladder1, err := t.probeQuotes(ctx, staticExtra.Contract, pairID, token0, token1, points0, points1, blockNumber)
	if err != nil {
		return p, err
	}

	extra := Extra{Ladder0: ladder0, Ladder1: ladder1}
	if len(ladder0) == 0 && len(ladder1) == 0 {
		extra.Unquoteable = true
		lg.Warnf("no valid direction at block %s", blockNumber)
	}

	return t.persist(p, extra, r0, r1, blockNumber), nil
}

func buildPoints(reserveIn *big.Int) []*big.Int {
	if reserveIn == nil || reserveIn.Sign() == 0 {
		return nil
	}
	grid := make([]*big.Int, 0, len(sampleBps))
	var last *big.Int
	for _, bps := range sampleBps {
		amt := new(big.Int).Mul(reserveIn, big.NewInt(int64(bps)))
		amt.Div(amt, big.NewInt(bpsDenominator))
		if amt.Sign() == 0 {
			continue
		}
		if last != nil && amt.Cmp(last) <= 0 {
			continue
		}
		grid = append(grid, amt)
		last = amt
	}
	return grid
}

func (t *PoolTracker) probeQuotes(
	ctx context.Context,
	contract string,
	pairID common.Hash,
	token0, token1 common.Address,
	grid0, grid1 []*big.Int,
	blockNumber *big.Int,
) ([]LadderPoint, []LadderPoint, error) {
	requests := make([]quoteCallArg, 0, len(grid0)+len(grid1))
	for _, amt := range grid0 {
		requests = append(requests, quoteCallArg{PairId: pairID, TokenIn: token0, TokenOut: token1, AmountIn: amt})
	}
	for _, amt := range grid1 {
		requests = append(requests, quoteCallArg{PairId: pairID, TokenIn: token1, TokenOut: token0, AmountIn: amt})
	}
	if len(requests) == 0 {
		return nil, nil, nil
	}

	var results []quoteCallResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    caliberABI,
		Target: contract,
		Method: methodBatchQuote,
		Params: []any{requests},
	}, []any{&results})
	if _, err := req.TryAggregate(); err != nil {
		return nil, nil, err
	}

	ladder0 := collectLadder(grid0, results, 0)
	ladder1 := collectLadder(grid1, results, len(grid0))
	return ladder0, ladder1, nil
}

func collectLadder(grid []*big.Int, results []quoteCallResult, offset int) []LadderPoint {
	ladder := make([]LadderPoint, 0, len(grid))
	for i, amt := range grid {
		idx := offset + i
		if idx >= len(results) {
			break
		}
		res := results[idx]
		if !res.Success || res.AmountOut == nil || res.AmountOut.Sign() <= 0 {
			continue
		}
		amtU, overflowIn := uint256.FromBig(amt)
		outU, overflowOut := uint256.FromBig(res.AmountOut)
		if overflowIn || overflowOut {
			continue
		}
		ladder = append(ladder, LadderPoint{AmountIn: amtU, AmountOut: outU})
	}
	return ladder
}

func (t *PoolTracker) persist(p entity.Pool, extra Extra, r0, r1 *uint256.Int, blockNumber *big.Int) entity.Pool {
	extraBytes, _ := json.Marshal(extra)
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{r0.Dec(), r1.Dec()}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p
}
