package bin

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (t *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := t.fetchRpcState(ctx, &p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}

func (t *PoolTracker) fetchRpcState(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	var result FetchRPCResult

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    binPoolManagerABI,
		Target: t.config.BinPoolManagerAddress,
		Method: binPoolManagerMethodGetSlot0,
		Params: []any{common.HexToHash(p.Address)},
	}, []any{&result.Slot0})

	_, err := rpcRequests.Aggregate()

	return &result, err
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pancake-infinity-bin pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		rpcData         *FetchRPCResult
		newPoolReserves entity.PoolReserves
		bins            []Bin
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = t.fetchRpcState(ctx, &p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")

		}

		return err
	})

	g.Go(func(context.Context) error {
		var err error

		bins, newPoolReserves, err = t.getBinsFromSubgraph(ctx, p.Address)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for bins")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		ProtocolFee: uint256.MustFromBig(rpcData.Slot0.ProtocolFee),
		LpFee:       uint256.MustFromBig(rpcData.Slot0.LpFee),
		ActiveBinID: uint32(rpcData.Slot0.ActiveId.Int64()),
		Bins:        bins,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Reserves = newPoolReserves
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) getBinsFromSubgraph(ctx context.Context, poolAddress string) ([]Bin, entity.PoolReserves, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	var (
		allowSubgraphError = t.config.IsAllowSubgraphError()

		lastBinId       int32 = -1
		unitX, unitY    *big.Float
		bins            []Bin
		newPoolReserves = entity.PoolReserves{"0", "0"}
		err             error
	)

	for {
		req := graphqlpkg.NewRequest(getBinsQuery(poolAddress, lastBinId, allowSubgraphError))

		var resp struct {
			Pair *LBPair `json:"lbpair"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			if allowSubgraphError {
				if resp.Pair == nil {
					l.WithFields(logger.Fields{
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Error("failed to query subgraph")

					return nil, entity.PoolReserves{}, err
				}
			} else {
				l.WithFields(logger.Fields{
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Error("failed to query subgraph")

				return nil, entity.PoolReserves{}, err
			}
		}

		if resp.Pair == nil || len(resp.Pair.Bins) == 0 {
			break
		}

		if unitX == nil {
			unitX, err = parseTokenDecimal(resp.Pair.TokenX.Decimals)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if unitY == nil {
			unitY, err = parseTokenDecimal(resp.Pair.TokenY.Decimals)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if newPoolReserves[0] != resp.Pair.ReserveX {
			newPoolReserves[0], err = parsePoolReserve(resp.Pair.ReserveX, unitX)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if newPoolReserves[1] != resp.Pair.ReserveY {
			newPoolReserves[1], err = parsePoolReserve(resp.Pair.ReserveY, unitY)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		subgraphBins := resp.Pair.Bins
		for _, subgraphBin := range subgraphBins {
			bin, err := transformSubgraphBin(subgraphBin, unitX, unitY)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}

			bins = append(bins, bin)
		}

		if len(subgraphBins) < graphFirstLimit {
			break
		}

		lastBinId = subgraphBins[len(subgraphBins)-1].BinID
	}

	sort.Slice(bins, func(i, j int) bool {
		return bins[i].ID < bins[j].ID
	})

	return bins, newPoolReserves, nil
}

func parseTokenDecimal(decimals string) (*big.Float, error) {
	decimalX, err := strconv.ParseInt(decimals, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token decimals: %w", err)
	}

	return bignumber.TenPowDecimals(uint8(decimalX)), nil
}

func parsePoolReserve(reserve string, unit *big.Float) (string, error) {
	reserveF, ok := new(big.Float).SetString(reserve)
	if !ok {
		return "", errors.New("can not convert pool's reserve from string to big.Float")
	}

	reserveInt, _ := new(big.Float).Mul(reserveF, unit).Int(nil)

	return reserveInt.String(), nil
}

func transformSubgraphBin(
	bin SubgraphBin,
	unitX *big.Float,
	unitY *big.Float,
) (Bin, error) {
	reserveX, ok := new(big.Float).SetString(bin.ReserveX)
	if !ok {
		return Bin{}, fmt.Errorf("[bin: %v] can not convert bin's reserveX from string to big.Float", bin.BinID)
	}
	reserveXInt, _ := new(big.Float).Mul(reserveX, unitX).Int(nil)

	reserveY, ok := new(big.Float).SetString(bin.ReserveY)
	if !ok {
		return Bin{}, fmt.Errorf("[bin: %v] can not convert bin's reserveY from string to big.Float", bin.BinID)
	}
	reserveYInt, _ := new(big.Float).Mul(reserveY, unitY).Int(nil)

	return Bin{
		ID:       uint32(bin.BinID),
		ReserveX: uint256.MustFromBig(reserveXInt),
		ReserveY: uint256.MustFromBig(reserveYInt),
	}, nil
}
