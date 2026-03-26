package feltir

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	log := logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": p.Address})

	reserves, blockNumber, err := t.fetchReserves(ctx, p)
	if err != nil {
		log.Errorf("feltir: fetchReserves failed: %v", err)
		return p, err
	}

	samples, err := t.fetchQuotes(ctx, p, reserves, blockNumber)
	if err != nil {
		log.Errorf("feltir: fetchQuotes failed: %v", err)
		return p, err
	}

	samples = filterSamples(samples, reserves)

	extraBytes, err := json.Marshal(Extra{Samples: samples})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) fetchReserves(ctx context.Context, p entity.Pool) ([2]*big.Int, *big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	var reserves [2]*big.Int
	vaultAddr := common.HexToAddress(t.cfg.MakerAddress)
	for i, tok := range p.Tokens {
		reserves[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: tok.Address,
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{vaultAddr},
		}, []any{&reserves[i]})
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return [2]*big.Int{}, nil, err
	}

	return reserves, res.BlockNumber, nil
}

func (t *PoolTracker) fetchQuotes(ctx context.Context, p entity.Pool, reserves [2]*big.Int, blockNumber *big.Int) ([2][][2]*big.Int, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	var samples [2][][2]*big.Int

	for i := range p.Tokens {
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		points := make([]*big.Int, 0, sampleSize+len(reserveSampleBps))

		dec := int(p.Tokens[i].Decimals)
		start := max(0, dec-sampleSize/2)
		for idx, k := 0, start; idx < sampleSize; idx, k = idx+1, k+1 {
			points = append(points, bignumber.TenPowInt(k))
		}

		if reserves[i] != nil && reserves[i].Sign() > 0 {
			for _, bps := range reserveSampleBps {
				pt := new(big.Int).Mul(reserves[i], big.NewInt(int64(bps)))
				pt.Div(pt, bignumber.BasisPoint)
				if pt.Sign() > 0 {
					points = append(points, pt)
				}
			}
		}

		sortAndDedup(&points)

		samples[i] = make([][2]*big.Int, len(points))
		for j, pt := range points {
			samples[i][j] = [2]*big.Int{new(big.Int).Set(pt), big.NewInt(0)}
			req.AddCall(&ethrpc.Call{
				ABI:    feltirABI,
				Target: t.cfg.FeltirAddress,
				Method: "getAmountOut",
				Params: []any{tokenIn, tokenOut, samples[i][j][0]},
			}, []any{&samples[i][j][1]})
		}
	}

	if _, err := req.TryBlockAndAggregate(); err != nil {
		return [2][][2]*big.Int{}, err
	}

	return samples, nil
}

func sortAndDedup(points *[]*big.Int) {
	s := *points
	if len(s) <= 1 {
		return
	}

	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Cmp(s[j-1]) < 0; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}

	result := s[:1]
	for _, v := range s[1:] {
		if v.Cmp(result[len(result)-1]) != 0 {
			result = append(result, v)
		}
	}

	*points = result
}

func filterSamples(samples [2][][2]*big.Int, reserves [2]*big.Int) [2][][2]*big.Int {
	for dir := range samples {
		outReserve := reserves[1-dir]
		var plateau *big.Int
		if outReserve != nil && outReserve.Sign() > 0 {
			plateau = new(big.Int).Sub(outReserve, bignumber.One)
		}

		valid := samples[dir][:0]
		plateauSeen := false
		for _, s := range samples[dir] {
			if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
				continue
			}

			if plateau != nil && s[1].Cmp(plateau) >= 0 {
				if plateauSeen {
					continue
				}
				plateauSeen = true
			}
			valid = append(valid, s)
		}
		samples[dir] = valid
	}
	return samples
}
