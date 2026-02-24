package kipseliprop

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
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
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{
					common.HexToAddress(p.Tokens[i].Address),
					samples[i][idx][0],
					common.HexToAddress(p.Tokens[1-i].Address),
				},
			}, []any{&samples[i][idx][1]})
			idx++
		}
	}

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

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

	extra := Extra{
		Samples: samples,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)

	tokenAddrs := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	var balances []*big.Int
	reqRes := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(res.BlockNumber)
	reqRes.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalances",
		Params: []any{tokenAddrs},
	}, []any{&balances})

	if _, err := reqRes.TryAggregate(); err != nil {
		return p, err
	}

	if len(balances) < 2 || balances[0] == nil || balances[1] == nil {
		return p, ErrInsufficientLiquidity
	}

	p.Reserves = []string{
		balances[0].String(),
		balances[1].String(),
	}

	p.Timestamp = time.Now().Unix()
	p.BlockNumber = res.BlockNumber.Uint64()

	return p, nil
}
