package kipseliprop

import (
	"context"
	"encoding/json"
	"math/big"
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

	samples, blockNumber, err := t.fetchQuotes(ctx, p, bTsMs)
	if err != nil {
		return p, err
	}

	t.warnGapInQuotes(p, samples)
	t.applyBuffer(samples)

	tokenAddrs := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	balances, caps, err := t.fetchBalancesAndCaps(ctx, blockNumber, tokenAddrs)
	if err != nil {
		return p, err
	}

	if len(balances) < 2 || balances[0] == nil || balances[1] == nil {
		return p, ErrInsufficientLiquidity
	}

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

func (t *PoolTracker) fetchQuotes(ctx context.Context, p entity.Pool, bTsMs *big.Int) ([][][2]*big.Int, *big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	samples := make([][][2]*big.Int, 2)

	msgTemplate := DomainType
	msgTemplate.Domain.ChainId = math.NewHexOrDecimal256(int64(t.cfg.ChainID))
	msgTemplate.Domain.VerifyingContract = hexutil.Encode(t.cfg.Verifier[:])

	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		sig, err := t.signQuoteMsg(msgTemplate, tokenIn, tokenOut, bTsMs)
		if err != nil {
			return nil, nil, err
		}

		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)
		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			samples[i][idx] = [2]*big.Int{bignumber.TenPowInt(k), new(big.Int)}
			req.AddCall(&ethrpc.Call{
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{tokenIn, samples[i][idx][0], tokenOut, bTsMs, sig},
			}, []any{&samples[i][idx][1]})
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return samples, res.BlockNumber, nil
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

func (t *PoolTracker) fetchBalancesAndCaps(ctx context.Context, blockNumber *big.Int, tokenAddrs []common.Address) ([]*big.Int, []*big.Int, error) {
	var balances, caps []*big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
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

	if _, err := req.TryAggregate(); err != nil {
		return nil, nil, err
	}
	return balances, caps, nil
}

func buildExtra(samples [][][2]*big.Int, balances, caps []*big.Int) Extra {
	extra := Extra{
		MaxIn: computeMaxIn(balances, caps),
	}

	for dir := range samples {
		maxIn := lo.Ternary(dir < len(extra.MaxIn), extra.MaxIn[dir], nil)
		extra.Samples = append(extra.Samples, filterSamples(samples[dir], maxIn))
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

func filterSamples(samples [][2]*big.Int, maxIn *big.Int) [][2]*big.Int {
	valid := samples[:0]
	for _, s := range samples {
		if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
			continue
		}
		if maxIn != nil && maxIn.Sign() > 0 && s[0].Cmp(maxIn) > 0 {
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
