package kipseliprop

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/crypto"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	signer       *crypto.Eip712Signer
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		signer:       crypto.NewEip712Signer(cfg.Quoter[:]),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	tsMs := time.Now().UnixMilli()
	bTsMs := big.NewInt(tsMs)

	tokenAddrs := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	balances, caps, blockNumber, err := t.fetchBalancesAndCaps(ctx, tokenAddrs)
	if err != nil {
		return p, err
	}

	if len(balances) < 2 || balances[0] == nil || balances[1] == nil {
		return p, ErrInsufficientLiquidity
	}

	maxIn := computeMaxIn(balances, caps)

	samples, err := t.fetchQuotes(ctx, p, bTsMs, maxIn, blockNumber)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, samples)
	t.applyBuffer(samples)

	p.Reserves = []string{balances[0].String(), balances[1].String()}

	extra := buildExtra(samples, balances, caps)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	bTsMs *big.Int,
	maxIn []*big.Int,
	blockNumber *big.Int,
) ([][][2]*big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	samples := make([][][2]*big.Int, 2)

	msgTemplate := DomainType
	msgTemplate.Domain.ChainId = math.NewHexOrDecimal256(int64(t.cfg.ChainID))
	msgTemplate.Domain.VerifyingContract = hexutil.Encode(t.cfg.Verifier[:])

	for i := range p.Tokens {
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		sig, err := t.signQuoteMsg(msgTemplate, tokenIn, tokenOut, bTsMs)
		if err != nil {
			return nil, err
		}

		points := make([]*big.Int, 0, sampleSize+len(maxInSampleBps))

		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)
		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			points = append(points, bignumber.TenPowInt(k))
		}

		var dirMaxIn *big.Int
		if i < len(maxIn) {
			dirMaxIn = maxIn[i]
		}
		if dirMaxIn != nil && dirMaxIn.Sign() > 0 {
			for _, bps := range maxInSampleBps {
				pt := new(big.Int).Mul(dirMaxIn, big.NewInt(int64(bps)))
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
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{tokenIn, samples[i][j][0], tokenOut, bTsMs, sig},
			}, []any{&samples[i][j][1]})
		}
	}

	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}

	return samples, nil
}

func (t *PoolTracker) signQuoteMsg(template apitypes.TypedData, tokenIn, tokenOut common.Address, bTsMs *big.Int) ([]byte, error) {
	msg := template
	msg.Message = apitypes.TypedDataMessage{
		"tokenIn":            [20]byte(tokenIn),
		"tokenOut":           [20]byte(tokenOut),
		"timestampInMilisec": bTsMs,
	}
	return t.signer.Sign(msg)
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

func (t *PoolTracker) fetchBalancesAndCaps(ctx context.Context, tokenAddrs []common.Address) ([]*big.Int, []*big.Int, *big.Int, error) {
	var balances, caps []*big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalances",
		Params: []any{tokenAddrs},
	}, []any{&balances})
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalanceCap",
		Params: []any{tokenAddrs},
	}, []any{&caps})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, nil, err
	}
	return balances, caps, res.BlockNumber, nil
}

func buildExtra(samples [][][2]*big.Int, balances, caps []*big.Int) Extra {
	extra := Extra{
		MaxIn: computeMaxIn(balances, caps),
	}

	for dir := range samples {
		maxIn := lo.Ternary(dir < len(extra.MaxIn), extra.MaxIn[dir], nil)
		outputReserve := lo.Ternary(1-dir < len(balances), balances[1-dir], nil)
		extra.Samples = append(extra.Samples, filterSamples(samples[dir], maxIn, outputReserve))
	}

	return extra
}

func computeMaxIn(balances, caps []*big.Int) []*big.Int {
	maxIn := make([]*big.Int, len(caps))
	for i, c := range caps {
		if c == nil || c.Sign() <= 0 || c.Cmp(bignumber.MaxUint256) == 0 {
			continue
		}
		if i < len(balances) && balances[i] != nil && c.Cmp(balances[i]) > 0 {
			maxIn[i] = new(big.Int).Sub(c, balances[i])
		}
	}
	return maxIn
}

func filterSamples(samples [][2]*big.Int, maxIn *big.Int, outputReserve *big.Int) [][2]*big.Int {
	valid := samples[:0]
	for _, s := range samples {
		if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
			continue
		}
		if maxIn != nil && maxIn.Sign() > 0 && s[0].Cmp(maxIn) > 0 {
			continue
		}
		if outputReserve != nil && outputReserve.Sign() > 0 && s[1].Cmp(outputReserve) >= 0 {
			continue
		}
		valid = append(valid, s)
	}
	return valid
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
					}).Warn("kipseli-prop quote gap detected (positive -> zero -> positive)")
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
