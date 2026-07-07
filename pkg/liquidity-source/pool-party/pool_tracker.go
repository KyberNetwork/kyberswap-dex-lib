package poolparty

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
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

	var rateToETH, balanceOf *big.Int

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

	req.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: p.Tokens[1].Address,
		Method: "balanceOf",
		Params: []any{
			common.HexToAddress(p.Address),
		},
	}, []any{&balanceOf})

	res, err := req.TryAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to track pool state from PoolParty")
		return p, err
	}

	poolStatus := poolStatusActive
	isVisible := true
	if balanceOf.Sign() == 0 {
		poolStatus = poolStatusCanceled
		isVisible = false
	}

	extra := Extra{
		PoolStatus:            poolStatus,
		IsVisible:             isVisible,
		BoostPriceBps:         d.config.BoostPriceBps,
		RateToETH:             rateToETH,
		PublicAmountAvailable: balanceOf,
		Exchange:              d.config.Exchange,
	}
	extraBytes, _ := json.Marshal(extra)

	p.Reserves[1] = balanceOf.String()

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	l.Info("successfully get new pool state")

	return p, nil
}
