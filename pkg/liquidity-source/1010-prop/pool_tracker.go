package prop

import (
	"context"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

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
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	balances, blockNumber, err := t.fetchBalances(ctx, staticExtra.RouterAddress, p.Tokens)
	if err != nil {
		return p, err
	}

	samples, err := t.fetchQuotes(ctx, p, staticExtra.RouterAddress, balances, blockNumber)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, samples)
	t.applyBuffer(samples)
	samples = filterSamples(samples)

	p.Reserves = []string{balances[0].String(), balances[1].String()}

	extra := Extra{Samples: samples}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

// fetchBalances calls getAssetReserves on the router and returns the two balances
// that correspond to the pool's token pair, in pool token order.
func (t *PoolTracker) fetchBalances(ctx context.Context, routerAddr string, tokens []*entity.PoolToken) ([]*big.Int, *big.Int, error) {
	var assetReserves AssetReserves
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: routerAddr,
		Method: "getAssetReserves",
		Params: nil,
	}, []any{&assetReserves})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	balanceByAddr := make(map[string]*big.Int, len(assetReserves.Tokens))
	for i, addr := range assetReserves.Tokens {
		if i < len(assetReserves.Balances) {
			balanceByAddr[strings.ToLower(hexutil.Encode(addr[:]))] = assetReserves.Balances[i]
		}
	}

	balances := make([]*big.Int, len(tokens))
	for i, tok := range tokens {
		bal := balanceByAddr[strings.ToLower(tok.Address)]
		if bal == nil {
			return nil, nil, ErrInsufficientLiquidity
		}
		balances[i] = bal
	}

	return balances, res.BlockNumber, nil
}

func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	routerAddr string,
	balances []*big.Int,
	blockNumber *big.Int,
) ([][][2]*big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	samples := make([][][2]*big.Int, 2)
	account := common.Address{}

	for i := range p.Tokens {
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		points := make([]*big.Int, 0, sampleSize+len(maxInSampleBps))

		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)
		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			points = append(points, bignumber.TenPowInt(k))
		}

		if i < len(balances) && balances[i] != nil && balances[i].Sign() > 0 {
			for _, bps := range maxInSampleBps {
				pt := new(big.Int).Mul(balances[i], big.NewInt(int64(bps)))
				pt.Div(pt, bignumber.BasisPoint)
				if pt.Sign() > 0 {
					points = append(points, pt)
				}
			}
		}

		sort.Slice(points, func(a, b int) bool {
			return points[a].Cmp(points[b]) < 0
		})
		points = dedupSorted(points)

		samples[i] = make([][2]*big.Int, len(points))
		for j, pt := range points {
			samples[i][j] = [2]*big.Int{new(big.Int).Set(pt), new(big.Int)}
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: routerAddr,
				Method: "quote",
				Params: []any{account, tokenIn, tokenOut, samples[i][j][0]},
			}, []any{&samples[i][j][1]})
		}
	}

	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}

	return samples, nil
}

func (t *PoolTracker) applyBuffer(samples [][][2]*big.Int) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for i := range samples {
		for j := range samples[i] {
			if s1 := samples[i][j][1]; s1 != nil {
				s1.Mul(s1, buf)
				s1.Div(s1, bignumber.BasisPoint)
			}
		}
	}
}

func filterSamples(samples [][][2]*big.Int) [][][2]*big.Int {
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
	return samples
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
					}).Warn("1010-prop quote gap detected (positive -> zero -> positive)")
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

func dedupSorted(sorted []*big.Int) []*big.Int {
	if len(sorted) <= 1 {
		return sorted
	}
	result := sorted[:1]
	for _, v := range sorted[1:] {
		if v.Cmp(result[len(result)-1]) != 0 {
			result = append(result, v)
		}
	}
	return result
}
