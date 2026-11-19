package dexv2

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	lensGuard    *ephemeral.Guard

	// readState and readTicks are the pool's state and tick reads, FetchRPCData and
	// queryTicksFromRPC unless a test stands in for the chain.
	readState func(ctx context.Context, p *entity.Pool, blockNumber uint64) (Extra, error)
	readTicks func(ctx context.Context, address string, tickIndexes []int, blockNumber uint64) ([]tickspkg.Tick,
		error)
}

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	_ *graphqlpkg.Client,
) *PoolTracker {
	t := &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		lensGuard:    ephemeral.NewGuard(config.LensGuard),
	}
	t.readState = t.FetchRPCData
	t.readTicks = t.queryTicksFromRPC
	return t
}

func (t *PoolTracker) BootstrapPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if t.config.UseLens {
		lensPool, _, err := t.bootstrapFromLens(ctx, p, params, nil)
		if err == nil {
			return lensPool, nil
		}
		// A node that cannot serve a lens read at all, as one ignoring a pinned block (HyperEVM's)
		// does for a read across blocks, has the pool bootstrapped without the lens rather than left
		// unsaved; any other failure is the bootstrap's own, to be tried again.
		if !ephemeral.Unreadable(err) {
			return lensPool, err
		}
		logger.WithFields(logger.Fields{"address": p.Address, "error": err}).Warn("failed to bootstrap the pool through the lens, bootstrapping it without")
		return t.bootstrapWithoutLens(ctx, p, params)
	}
	return t.getNewPoolState(ctx, p, params, nil)
}

// bootstrapWithoutLens bootstraps the pool without the lens: getNewPoolState reads its ticks from
// the bitmap at the block of its state, so they need no second read.
func (t *PoolTracker) bootstrapWithoutLens(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

// bootstrapFromLens reads the pool's state and ticks through the lens at block (nil for the
// latest), then the rest of its state, which the lens does not return, at the same block. It
// returns the pool with the block the lens read it at.
func (t *PoolTracker) bootstrapFromLens(ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams,
	block *big.Int) (entity.Pool, uint64, error) {
	snap, err := t.snapshot(ctx, p.Address, block)
	if err != nil {
		return entity.Pool{}, 0, err
	}

	extra, err := t.readState(ctx, &p, snap.block)
	if err != nil {
		return entity.Pool{}, 0, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, 0, err
	}
	if extra.Liquidity.Cmp(snap.liquidity) != 0 || extra.Tick.Int64() != int64(snap.tick) ||
		extra.SqrtPriceX96.Cmp(snap.sqrtPriceX96) != 0 || int(staticExtra.TickSpacing) != snap.tickSpacing {
		return entity.Pool{}, 0, fmt.Errorf("pool %s at block %d: the state read and the lens disagree",
			p.Address, snap.block)
	}

	extra.Ticks = snap.ticks
	// TicksBlock stays as it was: a log at or below the lens's block is read again whole (see
	// tickspkg.Holds), which holds one a reorg brought in, rather than dropped.
	extra.LensBlock = snap.block
	if p, err = withState(p, extra); err != nil {
		return entity.Pool{}, 0, err
	}
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(params.Logs).BlockTimestamp))
	p.BlockNumber = max(p.BlockNumber, snap.block, lo.LastOrEmpty(params.Logs).BlockNumber)
	return p, snap.block, nil
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

	var token0ExchangePricesAndConfig, token1ExchangePricesAndConfig, tokenReserves *big.Int
	var tokenBase [2]struct{ State TokenBaseState }
	var liquidityStatus *big.Int
	var userSupply, userBorrow [2]*big.Int

	token0 := lo.Ternary(
		staticExtra.IsNative[0],
		common.HexToAddress(valueobject.NativeAddress),
		common.HexToAddress(p.Tokens[0].Address),
	)
	token1 := lo.Ternary(
		staticExtra.IsNative[1],
		common.HexToAddress(valueobject.NativeAddress),
		common.HexToAddress(p.Tokens[1].Address),
	)
	dexKey := DexKey{
		Token0:      token0,
		Token1:      token1,
		Fee:         big.NewInt(int64(staticExtra.Fee)),
		TickSpacing: big.NewInt(int64(staticExtra.TickSpacing)),
		Controller:  common.HexToAddress(staticExtra.Controller),
	}

	req.AddCall(&ethrpc.Call{
		ABI:    abis.ResolverABI,
		Target: t.config.Resolver,
		Method: "getDexPoolState",
		Params: []any{big.NewInt(int64(dexType)), dexKey},
	}, []any{&res})

	token0Slot := calculateMappingStorageSlot(LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT, token0)
	req.AddCall(&ethrpc.Call{
		ABI:    abis.LiquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token0Slot},
	}, []any{&token0ExchangePricesAndConfig})

	token1Slot := calculateMappingStorageSlot(LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT, token1)
	req.AddCall(&ethrpc.Call{
		ABI:    abis.LiquidityABI,
		Target: t.config.Liquidity,
		Method: "readFromStorage",
		Params: []any{token1Slot},
	}, []any{&token1ExchangePricesAndConfig})

	tokenReserveSlot := calculateDoubleMappingStorageSlot(DEX_V2_TOKEN_RESERVES_MAPPING_SLOT, dexType, common.HexToHash(dexId))
	req.AddCall(&ethrpc.Call{
		ABI:    abis.DexV2ABI,
		Target: t.config.Dex,
		Method: "readFromStorage",
		Params: []any{tokenReserveSlot},
	}, []any{&tokenReserves})

	if staticExtra.TrackLiquidityLimits {
		req.AddCall(&ethrpc.Call{ABI: abis.LiquidityABI, Target: t.config.Liquidity, Method: "readFromStorage", Params: []any{common.BigToHash(big.NewInt(1))}}, []any{&liquidityStatus})
		for i, token := range []common.Address{token0, token1} {
			for j, slot := range []uint64{8, 9} {
				outer := calculateMappingStorageSlot(slot, common.HexToAddress(t.config.Dex))
				inner := crypto.Keccak256Hash(common.LeftPadBytes(token.Bytes(), 32), outer.Bytes())
				dst := &userSupply[i]
				if j == 1 {
					dst = &userBorrow[i]
				}
				req.AddCall(&ethrpc.Call{ABI: abis.LiquidityABI, Target: t.config.Liquidity, Method: "readFromStorage", Params: []any{inner}}, []any{dst})
			}
			req.AddCall(&ethrpc.Call{ABI: abis.ResolverABI, Target: t.config.Resolver,
				Method: "getTokenBaseState", Params: []any{token}}, []any{&tokenBase[i]})
		}
	}

	response, err := req.Aggregate()
	if err != nil {
		return Extra{}, err
	}
	if response.BlockNumber == nil {
		return Extra{}, fmt.Errorf("missing snapshot block")
	}
	timestamp, err := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(response.BlockNumber).GetCurrentBlockTimestamp()
	if err != nil {
		return Extra{}, err
	}

	extra := Extra{
		Liquidity:    res.DexPoolState.DexPoolStateRaw.DexVariables2Unpacked.ActiveLiquidity,
		SqrtPriceX96: res.DexPoolState.DexPoolStateRaw.DexVariablesUnpacked.CurrentSqrtPriceX96,
		Tick:         res.DexPoolState.DexPoolStateRaw.DexVariablesUnpacked.CurrentTick,

		DexVariables2:                 res.DexPoolState.DexPoolStateRaw.DexVariables2Packed,
		Token0ExchangePricesAndConfig: token0ExchangePricesAndConfig,
		Token1ExchangePricesAndConfig: token1ExchangePricesAndConfig,

		TokenReserves: tokenReserves,
		SnapshotBlock: response.BlockNumber.Uint64(), SnapshotTimestamp: timestamp,
		SupplyOnlySettlement: t.config.SupplyOnlySettlement,
	}

	if staticExtra.TrackLiquidityLimits {
		extra.FloatBalance = [2]*big.Int{tokenBase[0].State.FloatBalance, tokenBase[1].State.FloatBalance}
		extra.MaxCap = [2]*big.Int{tokenBase[0].State.MaxCap, tokenBase[1].State.MaxCap}
		extra.UnaccountedBorrow = [2]*big.Int{tokenBase[0].State.UnaccountedBorrow, tokenBase[1].State.UnaccountedBorrow}
		extra.SwapOutWithdraw = [2]*big.Int{tokenBase[0].State.AvailableForSwapOutWithdraw, tokenBase[1].State.AvailableForSwapOutWithdraw}
		extra.SwapOutBorrow = [2]*big.Int{tokenBase[0].State.AvailableForSwapOutBorrow, tokenBase[1].State.AvailableForSwapOutBorrow}
		if liquidityStatus == nil {
			return Extra{}, ErrUnsupportedLiquidityState
		}
		for i := range tokenBase {
			b := tokenBase[i].State
			config := token0ExchangePricesAndConfig
			if i == 1 {
				config = token1ExchangePricesAndConfig
			}
			allowed, permissionErr := liquidityPermissions(b, userSupply[i], userBorrow[i], config, liquidityStatus, timestamp)
			if permissionErr != nil {
				return Extra{}, permissionErr
			}
			extra.LiquidityDebt[i] = b.DexBorrowAtLiquidity.Borrow
			extra.WithdrawPaused[i] = b.DexSupplyAtLiquidity.IsPaused || b.DexSupplyAtLiquidity.IsWithdrawPaused || b.LiquidityTokenData.IsPaused || b.LiquidityTokenData.IsWithdrawPaused
			extra.SupplyPaused[i] = b.DexSupplyAtLiquidity.IsPaused || b.DexSupplyAtLiquidity.IsSupplyPaused || b.LiquidityTokenData.IsPaused || b.LiquidityTokenData.IsSupplyPaused
			extra.PaybackPaused[i] = b.DexBorrowAtLiquidity.IsPaused || b.DexBorrowAtLiquidity.IsPaybackPaused || b.LiquidityTokenData.IsPaused || b.LiquidityTokenData.IsPaybackPaused
			extra.BorrowPaused[i] = b.DexBorrowAtLiquidity.IsPaused || b.DexBorrowAtLiquidity.IsBorrowPaused || b.LiquidityTokenData.IsPaused || b.LiquidityTokenData.IsBorrowPaused
			var limitErr error
			extra.LiquidityLimits[i], limitErr = liquidityLimitsFromRPC(b, allowed)
			if limitErr != nil {
				return Extra{}, limitErr
			}
		}
	}
	return extra, nil
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[%s] Start getting new state of pool %v", t.config.DexID, p.Address)

	extra, err := t.fetchRPCData(ctx, p, 0, overrides)
	if err != nil {
		return entity.Pool{}, err
	}

	ticks, err := t.fetchPoolTicksFromBitmap(ctx, p, extra.SnapshotBlock)
	if err != nil {
		return entity.Pool{}, err
	}
	extra.Ticks = ticks
	extra.TicksBlock = extra.SnapshotBlock

	if p, err = withState(p, extra); err != nil {
		return entity.Pool{}, err
	}
	p.BlockNumber = extra.SnapshotBlock
	p.Timestamp = int64(extra.SnapshotTimestamp)

	logger.Infof("[%s] Finish getting new state of pool %v", t.config.DexID, p.Address)

	return p, nil
}

// withState writes extra into p, with the reserves it holds converted to token amounts. Every field
// of extra but the ticks comes from one state read: the simulator takes the fee, its dynamic part
// and the pool accounting flag from DexVariables2 and checks swaps against TokenReserves, so none
// of them may be kept from an older read.
func withState(p entity.Pool, extra Extra) (entity.Pool, error) {
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	reserve0Adjusted, reserve1Adjusted := extractTokenReserves(extra.TokenReserves)
	c, err := _calculateVars(extra.DexVariables2, extra.Token0ExchangePricesAndConfig, extra.Token1ExchangePricesAndConfig, extra.SnapshotTimestamp)
	if err != nil {
		return entity.Pool{}, err
	}

	_, dexType := parseFluidDexV2PoolAddress(p.Address)
	reserve0 := adjustedToAmount(reserve0Adjusted, c.Token0NumeratorPrecision, c.Token0DenominatorPrecision,
		lo.Ternary(dexType == D3_MODULE, c.Token0SupplyExchangePrice, c.Token0BorrowExchangePrice))
	reserve1 := adjustedToAmount(reserve1Adjusted, c.Token1NumeratorPrecision, c.Token1DenominatorPrecision,
		lo.Ternary(dexType == D3_MODULE, c.Token1SupplyExchangePrice, c.Token1BorrowExchangePrice))

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	return p, nil
}
