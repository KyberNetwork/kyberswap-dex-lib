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
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	tsMs := time.Now().UnixMilli()
	bTsMs := big.NewInt(tsMs)
	samples := make([][][2]*big.Int, 2)
	typedMsgTemplate := DomainType
	typedMsgTemplate.Domain.ChainId = math.NewHexOrDecimal256(int64(t.cfg.ChainID))
	typedMsgTemplate.Domain.VerifyingContract = hexutil.Encode(t.cfg.Verifier[:])
	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		start := lo.Ternary(p.Tokens[i].Decimals < sampleSize/2, 0, p.Tokens[i].Decimals-sampleSize/2)
		idx := 0
		tokenIn, tokenOut := common.HexToAddress(p.Tokens[i].Address), common.HexToAddress(p.Tokens[1-i].Address)
		typedMsg := typedMsgTemplate
		typedMsg.Message = apitypes.TypedDataMessage{
			"tokenIn":            [20]byte(tokenIn),
			"tokenOut":           [20]byte(tokenOut),
			"timestampInMilisec": bTsMs,
		}
		sig, err := t.signer.Sign(typedMsg)
		if err != nil {
			return p, err
		}
		for k := start; k <= start+sampleSize-1 && idx < sampleSize; k++ {
			samples[i][idx] = [2]*big.Int{bignumber.TenPowInt(k), new(big.Int)}
			req.AddCall(&ethrpc.Call{
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{
					tokenIn,
					samples[i][idx][0],
					tokenOut,
					bTsMs,
					sig,
				},
			}, []any{&samples[i][idx][1]})
			idx++
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

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

	tokenAddrs := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	var balances []*big.Int
	var caps []*big.Int
	reqRes := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	reqRes.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalances",
		Params: []any{tokenAddrs},
	}, []any{&balances})
	reqRes.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalanceCap",
		Params: []any{tokenAddrs},
	}, []any{&caps})

	if _, err := reqRes.TryAggregate(); err != nil {
		return p, err
	}

	if len(balances) < 2 || balances[0] == nil || balances[1] == nil {
		return p, ErrInsufficientLiquidity
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

	p.Reserves = []string{
		balances[0].String(),
		balances[1].String(),
	}

	extra := Extra{
		Samples: samples,
	}
	extra.MaxIn = make([]*big.Int, len(caps))
	for i, c := range caps {
		if c == nil || c.Sign() <= 0 || c.Cmp(bignumber.MaxUint256) == 0 {
			continue
		}
		if i < len(balances) && balances[i] != nil && c.Cmp(balances[i]) > 0 {
			extra.MaxIn[i] = new(big.Int).Sub(c, balances[i])
		}
	}
	for dir := range samples {
		maxIn := lo.Ternary(dir < len(extra.MaxIn), extra.MaxIn[dir], nil)
		if maxIn == nil || maxIn.Sign() <= 0 {
			continue
		}
		valid := samples[dir][:0]
		for _, s := range samples[dir] {
			if s[0] != nil && s[0].Cmp(maxIn) <= 0 {
				valid = append(valid, s)
			}
		}
		samples[dir] = valid
	}

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
