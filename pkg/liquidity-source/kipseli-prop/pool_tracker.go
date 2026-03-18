package kipseliprop

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	propamm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/prop-amm"
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
	var prevExtra Extra
	if p.Extra != "" {
		_ = json.Unmarshal([]byte(p.Extra), &prevExtra)
	}

	// Round 1: query on-chain quotes at chosen sample points.
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	tsMs := big.NewInt(time.Now().UnixMilli())
	samples := make([][][2]*big.Int, len(p.Tokens))
	incremental := make([]bool, len(p.Tokens))

	for i := range p.Tokens {
		prevMin, prevMax := propamm.ValidRangeFromSamples(prevExtra.Samples, i)
		incremental[i] = prevMin != nil && prevMax != nil && prevMax.Cmp(prevMin) > 0

		tokenIn, tokenOut := common.HexToAddress(p.Tokens[i].Address), common.HexToAddress(p.Tokens[1-i].Address)
		sig := t.signQuote(tokenIn, tokenOut, tsMs)

		for _, amt := range propamm.BuildQueryPoints(p.Tokens[i].Decimals, prevMin, prevMax) {
			samples[i] = append(samples[i], [2]*big.Int{amt, new(big.Int)})
			req.AddCall(&ethrpc.Call{
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{tokenIn, amt, tokenOut, tsMs, sig},
			}, []any{&samples[i][len(samples[i])-1][1]})
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	propamm.ApplyBuffer(samples, t.cfg.Buffer)

	// Round 2 (cold start only): refine cap boundary.
	if err := t.refineColdStartCap(ctx, res.BlockNumber, p, samples, incremental, tsMs); err != nil {
		return p, err
	}

	for i := range samples {
		samples[i] = propamm.CleanSamples(samples[i])
	}

	// Fetch reserves and caps.
	tokenAddrs := lo.Map(p.Tokens, func(tok *entity.PoolToken, _ int) common.Address {
		return common.HexToAddress(tok.Address)
	})
	var balances, caps []*big.Int
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

	extra := Extra{Samples: samples}
	extra.Caps = lo.Map(caps, func(c *big.Int, _ int) *big.Int {
		if c == nil || c.Sign() <= 0 || c.Cmp(bignumber.MaxUint256) == 0 {
			return nil
		}
		return c
	})

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{balances[0].String(), balances[1].String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = res.BlockNumber.Uint64()

	return p, nil
}

// refineColdStartCap is kipseli-specific: uses EIP-712 signed `quote` call on router.
func (t *PoolTracker) refineColdStartCap(
	ctx context.Context, blockNumber *big.Int, p entity.Pool,
	samples [][][2]*big.Int, incremental []bool, tsMs *big.Int,
) error {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	refined := make([][][2]*big.Int, len(samples))
	hasCall := false

	for dir := range samples {
		if dir < len(incremental) && incremental[dir] {
			continue
		}
		points := propamm.RefineCapPoints(propamm.FindCapBoundary(samples[dir]))
		if len(points) == 0 {
			continue
		}

		tokenIn, tokenOut := common.HexToAddress(p.Tokens[dir].Address), common.HexToAddress(p.Tokens[1-dir].Address)
		sig := t.signQuote(tokenIn, tokenOut, tsMs)

		refined[dir] = lo.Map(points, func(amt *big.Int, _ int) [2]*big.Int {
			return [2]*big.Int{amt, new(big.Int)}
		})
		for j := range refined[dir] {
			hasCall = true
			req.AddCall(&ethrpc.Call{
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{tokenIn, refined[dir][j][0], tokenOut, tsMs, sig},
			}, []any{&refined[dir][j][1]})
		}
	}

	if !hasCall {
		return nil
	}
	if _, err := req.TryAggregate(); err != nil {
		return err
	}
	for dir := range refined {
		samples[dir] = append(samples[dir], lo.Filter(refined[dir], func(r [2]*big.Int, _ int) bool {
			return r[0] != nil && r[1] != nil && r[1].Sign() > 0
		})...)
	}
	return nil
}

func (t *PoolTracker) signQuote(tokenIn, tokenOut common.Address, tsMs *big.Int) []byte {
	typedMsg := DomainType
	typedMsg.Domain.ChainId = math.NewHexOrDecimal256(int64(t.cfg.ChainID))
	typedMsg.Domain.VerifyingContract = hexutil.Encode(t.cfg.Verifier[:])
	typedMsg.Message = apitypes.TypedDataMessage{
		"tokenIn":            [20]byte(tokenIn),
		"tokenOut":           [20]byte(tokenOut),
		"timestampInMilisec": tsMs,
	}
	sig, _ := t.signer.Sign(typedMsg)
	return sig
}
