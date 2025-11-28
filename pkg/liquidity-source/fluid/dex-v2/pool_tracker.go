package dexv2

import (
	"context"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
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

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool %v", t.config.DexID, p.Address)

	dexId, dexType := parseFluidDexV2PoolAddress(p.Address)

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return entity.Pool{}, err
	}

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)

	// At first, I use `var dexPoolState DexPoolState` here, but it causes
	// error `abi: cannot unmarshal struct {...} in to *big.Int`.
	// After tracing, I found that if the output is a struct, go-ethereum
	// only pick the first field of the struct to unmarshal the data (don't know why).
	// https://github.com/ethereum/go-ethereum/blob/3bbf5f5b6a9cd5ba998f6580586ddf208217e915/accounts/abi/argument.go#L135-L137
	// So I create a wrapper struct to hold the DexPoolState struct.
	var res struct {
		DexPoolState DexPoolState
	}

	var token0ExchangePricesAndConfig, token1ExchangePricesAndConfig *big.Int

	req.AddCall(&ethrpc.Call{
		ABI:    resolverABI,
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
			extra.IsNative[0],
			common.HexToAddress(valueobject.NativeAddress),
			common.HexToAddress(p.Tokens[0].Address),
		),
	)
	req.AddCall(&ethrpc.Call{
		ABI:    liquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token0Slot},
	}, []any{&token0ExchangePricesAndConfig})

	token1Slot := calculateMappingStorageSlot(
		LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT,
		lo.Ternary(
			extra.IsNative[1],
			common.HexToAddress(valueobject.NativeAddress),
			common.HexToAddress(p.Tokens[1].Address),
		),
	)
	req.AddCall(&ethrpc.Call{
		ABI:    liquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token1Slot},
	}, []any{&token1ExchangePricesAndConfig})

	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	extra.DexVariables = res.DexPoolState.DexVariablesUnpacked
	extra.DexVariables2 = res.DexPoolState.DexVariables2Unpacked

	extra.Token0ExchangePricesAndConfig = token0ExchangePricesAndConfig
	extra.Token1ExchangePricesAndConfig = token1ExchangePricesAndConfig

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

	var reserve0, reserve1 big.Int
	if extra.DexVariables.CurrentSqrtPriceX96.Sign() != 0 {
		// reserve0 = liquidity / sqrtPriceX96 * Q96
		reserve0.Mul(extra.DexVariables2.ActiveLiquidity, uniswapv4.Q96)
		reserve0.Div(&reserve0, extra.DexVariables.CurrentSqrtPriceX96)
	}
	// reserve1 = liquidity * sqrtPriceX96 / Q96
	reserve1.Mul(extra.DexVariables2.ActiveLiquidity, extra.DexVariables.CurrentSqrtPriceX96)
	reserve1.Div(&reserve1, uniswapv4.Q96)
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
