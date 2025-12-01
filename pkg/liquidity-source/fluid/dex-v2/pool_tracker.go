package dexv2

import (
	"context"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
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

func (t *PoolTracker) fetchRPCData(
	ctx context.Context,
	p entity.Pool,
	blockNumber uint64,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, error) {
	dexId, dexType := parseFluidDexV2PoolAddress(p.Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return Extra{}, err
	}

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		req.SetBlockNumber(&blockNumberBI)
	}

	var res struct {
		DexPoolState DexPoolState
	}

	var token0ExchangePricesAndConfig, token1ExchangePricesAndConfig *big.Int

	req.AddCall(&ethrpc.Call{
		ABI:    abis.ResolverABI,
		Target: t.config.Resolver,
		Method: "getDexPoolState",
		Params: []any{
			big.NewInt(int64(dexType)),
			common.HexToHash(dexId),
		},
	}, []any{&res})

	token0Slot := calculateMappingStorageSlot(
		LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT,
		lo.Ternary(
			staticExtra.IsNative[0],
			common.HexToAddress(valueobject.NativeAddress),
			common.HexToAddress(p.Tokens[0].Address),
		),
	)
	req.AddCall(&ethrpc.Call{
		ABI:    abis.LiquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token0Slot},
	}, []any{&token0ExchangePricesAndConfig})

	token1Slot := calculateMappingStorageSlot(
		LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT,
		lo.Ternary(
			staticExtra.IsNative[1],
			common.HexToAddress(valueobject.NativeAddress),
			common.HexToAddress(p.Tokens[1].Address),
		),
	)
	req.AddCall(&ethrpc.Call{
		ABI:    abis.LiquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token1Slot},
	}, []any{&token1ExchangePricesAndConfig})

	if _, err := req.Aggregate(); err != nil {
		return Extra{}, err
	}

	extra := Extra{
		Liquidity:    res.DexPoolState.DexVariables2Unpacked.ActiveLiquidity,
		SqrtPriceX96: res.DexPoolState.DexVariablesUnpacked.CurrentSqrtPriceX96,
		Tick:         res.DexPoolState.DexVariablesUnpacked.CurrentTick,

		Token0ExchangePricesAndConfig: token0ExchangePricesAndConfig,
		Token1ExchangePricesAndConfig: token1ExchangePricesAndConfig,
	}

	return extra, nil
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool %v", t.config.DexID, p.Address)

	extra, err := t.fetchRPCData(ctx, p, 0, overrides)
	if err != nil {
		return entity.Pool{}, err
	}

	ticks, err := t.fetchPoolTicksFromSubgraph(ctx, p)
	if err != nil {
		return entity.Pool{}, err
	}
	extra.Ticks = ticks

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	reserve0, reserve1, err := calculateReservesFromTicks(extra.SqrtPriceX96, ticks)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}

	logger.Infof("[%s] Finish getting new state of pool %v", t.config.DexID, p.Address)

	return p, nil
}

func (t *PoolTracker) fetchPoolTicksFromSubgraph(
	ctx context.Context,
	p entity.Pool,
) ([]Tick, error) {
	dexId, dexType := parseFluidDexV2PoolAddress(p.Address)

	dexTypeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(dexTypeBytes, uint32(dexType))

	dexIdBytes := common.Hex2Bytes(dexId[2:])

	poolIdBytes := append(dexIdBytes, dexTypeBytes...)
	poolId := "0x" + common.Bytes2Hex(poolIdBytes)

	lastTickIdx := 0
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(poolId, lastTickIdx))

		var resp struct {
			Ticks []TickResp `json:"ticks"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			return nil, err
		}

		if len(resp.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Ticks...)

		if len(resp.Ticks) < graphFirstLimit {
			break
		}

		lastTickIdx = resp.Ticks[len(resp.Ticks)-1].Tick
	}

	return lo.Map(ticks, func(tick TickResp, _ int) Tick {
		return Tick{
			Index:          tick.Tick,
			LiquidityNet:   bignumber.NewBig10(tick.LiquidityNet),
			LiquidityGross: bignumber.NewBig10(tick.LiquidityGross),
		}
	}), nil
}
