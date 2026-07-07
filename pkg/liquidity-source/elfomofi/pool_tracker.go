package elfomofi

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	samples := make([][][2]*big.Int, len(p.Tokens))
	for i := range p.Tokens {
		samples[i] = make([][2]*big.Int, sampleSize)
		start := lo.Ternary(p.Tokens[i].Decimals < sampleSize/2, 0, p.Tokens[i].Decimals-sampleSize/2)
		index := 0
		for k := start; k < start+sampleSize; k++ {
			samples[i][index] = [2]*big.Int{bignumber.TenPowInt(k), new(big.Int)}
			req.AddCall(&ethrpc.Call{
				ABI:    factoryABI,
				Target: t.config.FactoryAddress,
				Method: "getAmountOut",
				Params: []any{
					common.HexToAddress(p.Tokens[i].Address),
					common.HexToAddress(p.Tokens[1-i].Address),
					samples[i][index][0],
				},
			}, []any{&samples[i][index][1]})
			index++
		}
	}

	_, err := req.TryAggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	// Scale samples with buffer
	buffer := big.NewInt(t.config.Buffer)
	for i := range samples {
		for j := range samples[i] {
			if samples[i][j][1] != nil {
				samples[i][j][1].Mul(samples[i][j][1], buffer).Div(samples[i][j][1], bignumber.BasisPoint)
			}
		}
	}

	// Turn the cumulative (amountIn, amountOut) samples into marginal order-book price levels: each
	// level's price is the marginal rate for that increment only, so consuming a level shifts later
	// quotes to the next (worse) rate instead of reusing one bracket's average rate for its whole range.
	var extra orderbook.Extra
	var reserves [2]big.Int
	for i := range samples {
		decIn, decOut := math.Pow10(int(p.Tokens[i].Decimals)), math.Pow10(int(p.Tokens[1-i].Decimals))

		levels := make([]orderbook.Level, 1, sampleSize+1) // first level == min trade == 0
		var prevIn, prevOut float64
		var prevOutWei big.Int
		for _, sample := range samples[i] {
			outWei := sample[1]
			if outWei == nil || outWei.Cmp(&prevOutWei) <= 0 {
				break
			}

			inF, _ := sample[0].Float64()
			outF, _ := outWei.Float64()
			in, out := inF/decIn, outF/decOut
			size := in - prevIn
			levels = append(levels, orderbook.Level{size, (out - prevOut) / size})

			prevIn, prevOut = in, out
			prevOutWei = *outWei
		}
		extra.LevelsFrom[i] = levels

		// The largest cumulative amountOut reached before liquidity ran out is the max reserve
		// obtainable on the other side.
		reserves[1-i] = prevOutWei
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{reserves[0].String(), reserves[1].String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
