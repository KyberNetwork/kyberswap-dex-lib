package poolparty

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphql.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	var rateToETH *big.Int
	var blockNumber big.Int
	var subgraphPool SubgraphPool

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		req := d.ethrpcClient.NewRequest().SetContext(ctx)

		req.AddCall(&ethrpc.Call{
			ABI:    oneInchOracle,
			Target: d.config.Oracle,
			Method: "getRateToEth",
			Params: []any{
				common.HexToAddress(p.Tokens[1].Address), // srcToken (address)
				false,                                    // useSrcWrappers (bool)
			},
		}, []any{&rateToETH})

		res, err := req.TryAggregate()
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to getRateToETH")
			return err
		}

		if res.BlockNumber != nil {
			blockNumber.Set(res.BlockNumber)
		}

		return nil
	})

	g.Go(func(ctx context.Context) error {
		req := graphql.NewRequest(getPoolState(p.Address))

		var res struct {
			Pool SubgraphPool `json:"pool"`
		}

		if err := d.graphqlClient.Run(ctx, req, &res); err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for pool state")
			return err
		}

		subgraphPool = res.Pool

		return nil
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	var prevExtra Extra
	if err := json.Unmarshal([]byte(p.Extra), &prevExtra); err != nil {
		return p, err
	}

	newPublicAmountAvailable := lo.Ternary(
		subgraphPool.PublicAmountAvailable != "",
		bignumber.NewBig10(subgraphPool.PublicAmountAvailable),
		prevExtra.PublicAmountAvailable,
	)

	extra := Extra{
		PoolStatus:            subgraphPool.PoolStatus,
		IsVisible:             subgraphPool.IsVisible,
		BoostPriceBps:         d.config.BoostPriceBps,
		RateToETH:             rateToETH,
		PublicAmountAvailable: newPublicAmountAvailable,
		Exchange:              d.config.Exchange,
	}
	extraBytes, _ := json.Marshal(extra)

	if extra.PoolStatus != poolStatusActive || !extra.IsVisible {
		p.Reserves[0] = "0"
	} else {
		p.Reserves[1] = subgraphPool.PublicAmountAvailable
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}
