package lunya

import (
	"context"
	"math/big"
	"math/bits"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState reads a pool's state at the latest block and pins every other read to that block.
// Scalars are refreshed even without logs: a dynamic fee moves with time, not only with swaps, and so
// does a STABLE pool's amplification ramp.
//
// For a CL or CP pool the first pass also walks the whole tick tree; after that only the ticks named by
// the Mint/Burn logs in params are re-read, which also covers reorgs since nothing is replayed from
// logs. A STABLE pool has no ticks, so every pass is a full refresh.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"dexId": t.config.DexID, "address": p.Address})

	var staticExtra StaticExtra
	if len(p.StaticExtra) > 0 {
		if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
			return p, err
		}
	}

	var extra Extra
	if len(p.Extra) > 0 {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			return p, err
		}
	}

	var (
		newExtra           Extra
		reserve0, reserve1 *big.Int
		blockNumber        *big.Int
		err                error
	)
	if staticExtra.PoolType == poolTypeStable {
		newExtra, reserve0, reserve1, blockNumber, err = t.fetchStableState(ctx, p)
	} else {
		newExtra, reserve0, reserve1, blockNumber, err = t.fetchCLState(ctx, p, extra, params.Logs)
	}
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to fetch pool state")
		return p, err
	}

	extraBytes, err := json.Marshal(newExtra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = float64(newExtra.Fee)
	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) fetchCLState(ctx context.Context, p entity.Pool, previous Extra,
	logs []ethtypes.Log) (Extra, *big.Int, *big.Int, *big.Int, error) {
	var (
		slot0        slot0Resp
		liquidity    *big.Int
		feeToken     uint8
		plugin       common.Address
		pluginConfig uint16
		tickTreeRoot uint32
		reserve0     *big.Int
		reserve1     *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	for _, c := range []struct {
		method string
		out    any
	}{
		{poolMethodSlot0, &slot0},
		{poolMethodLiquidity, &liquidity},
		{poolMethodFeeToken, &feeToken},
		{poolMethodPlugin, &plugin},
		{poolMethodPluginConfig, &pluginConfig},
		{poolMethodTickTreeRoot, &tickTreeRoot},
	} {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: c.method}, []any{c.out})
	}
	t.addReserveCalls(req, p, &reserve0, &reserve1)

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return Extra{}, nil, nil, nil, err
	}
	if slices.Contains(resp.Result, false) || resp.BlockNumber == nil {
		return Extra{}, nil, nil, nil, ErrFailedCall
	}

	fee, halted, err := t.fetchPluginState(ctx, plugin, pluginConfig, uint32(slot0.Fee.Uint64()), resp.BlockNumber)
	if err != nil {
		return Extra{}, nil, nil, nil, err
	}

	var ticks []Tick
	if previous.SqrtPriceX96 == nil {
		ticks, err = t.fetchAllTicks(ctx, p.Address, tickTreeRoot, resp.BlockNumber)
	} else {
		ticks, err = t.refreshTicks(ctx, p.Address, previous.Ticks, touchedTicks(p.Address, logs), resp.BlockNumber)
	}
	if err != nil {
		return Extra{}, nil, nil, nil, err
	}

	return Extra{
		SqrtPriceX96: uint256.MustFromBig(slot0.SqrtPriceX96),
		Liquidity:    uint256.MustFromBig(liquidity),
		Fee:          fee,
		FeeToken:     feeToken,
		Halted:       halted,
		Tick:         int(slot0.Tick.Int64()),
		Ticks:        ticks,
	}, reserve0, reserve1, resp.BlockNumber, nil
}

func (t *PoolTracker) fetchStableState(ctx context.Context, p entity.Pool) (Extra, *big.Int, *big.Int, *big.Int,
	error) {
	var (
		slot0             slot0Resp
		liquidity         *big.Int
		feeToken          uint8
		plugin            common.Address
		pluginConfig      uint16
		curveReserve0     *big.Int
		curveReserve1     *big.Int
		rate0             *big.Int
		rate1             *big.Int
		priceScaleSqrtQ96 *big.Int
		amplificationX100 uint32
		ramping           bool
		ramp              amplificationRampResp
		reserve0          *big.Int
		reserve1          *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	for _, c := range []struct {
		method string
		out    any
	}{
		{poolMethodSlot0, &slot0},
		{poolMethodLiquidity, &liquidity},
		{poolMethodFeeToken, &feeToken},
		{poolMethodPlugin, &plugin},
		{poolMethodPluginConfig, &pluginConfig},
		{poolMethodCurveReserve0, &curveReserve0},
		{poolMethodCurveReserve1, &curveReserve1},
		{poolMethodRate0, &rate0},
		{poolMethodRate1, &rate1},
		{poolMethodPriceScaleSqrtQ96, &priceScaleSqrtQ96},
		{poolMethodAmplificationX100, &amplificationX100},
		{poolMethodAmplificationRamping, &ramping},
		{poolMethodAmplificationRamp, &ramp},
	} {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: c.method}, []any{c.out})
	}
	t.addReserveCalls(req, p, &reserve0, &reserve1)

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return Extra{}, nil, nil, nil, err
	}
	if slices.Contains(resp.Result, false) || resp.BlockNumber == nil {
		return Extra{}, nil, nil, nil, ErrFailedCall
	}

	fee, halted, err := t.fetchPluginState(ctx, plugin, pluginConfig, uint32(slot0.Fee.Uint64()), resp.BlockNumber)
	if err != nil {
		return Extra{}, nil, nil, nil, err
	}

	extra := Extra{
		SqrtPriceX96:      uint256.MustFromBig(slot0.SqrtPriceX96),
		Liquidity:         uint256.MustFromBig(liquidity),
		Fee:               fee,
		FeeToken:          feeToken,
		Halted:            halted,
		CurveReserve0:     uint256.MustFromBig(curveReserve0),
		CurveReserve1:     uint256.MustFromBig(curveReserve1),
		Rate0:             uint256.MustFromBig(rate0),
		Rate1:             uint256.MustFromBig(rate1),
		PriceScaleSqrtQ96: uint256.MustFromBig(priceScaleSqrtQ96),
		AmplificationX100: amplificationX100,
	}
	if ramping {
		extra.Ramp = &AmplificationRamp{
			StartAmplification:  ramp.StartAmplification,
			TargetAmplification: ramp.TargetAmplification,
			StartTime:           ramp.StartTime,
			EndTime:             ramp.EndTime,
		}
	}

	return extra, reserve0, reserve1, resp.BlockNumber, nil
}

func (t *PoolTracker) addReserveCalls(req *ethrpc.Request, p entity.Pool, reserve0, reserve1 **big.Int) {
	for i, out := range []**big.Int{reserve0, reserve1} {
		req.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[i].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{out})
	}
}

// fetchPluginState resolves the fee and the halt the way a pool's swap meets them: beforeSwap runs under
// either flag, its return is the fee only under DYNAMIC_FEE, and LunyaDefaultPlugin reverts it unless
// SecurityModule is Active. currentFee() is a view over the plugin's oracle, so it is read as it is.
func (t *PoolTracker) fetchPluginState(ctx context.Context, plugin common.Address, pluginConfig uint16,
	slot0Fee uint32, blockNumber *big.Int) (fee uint32, halted bool, err error) {
	if plugin == (common.Address{}) || pluginConfig&(pluginFlagBeforeSwap|pluginFlagDynamicFee) == 0 {
		return slot0Fee, false, nil
	}

	var (
		currentFee *big.Int
		status     uint8
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req.AddCall(&ethrpc.Call{ABI: pluginABI, Target: plugin.Hex(), Method: pluginMethodCurrentFee},
		[]any{&currentFee})
	req.AddCall(&ethrpc.Call{ABI: pluginABI, Target: plugin.Hex(), Method: pluginMethodStatus}, []any{&status})
	resp, err := req.TryAggregate()
	if err != nil {
		return 0, false, err
	}

	fee = slot0Fee
	if pluginConfig&pluginFlagDynamicFee != 0 {
		if !resp.Result[0] || currentFee == nil {
			return 0, false, ErrFailedCall
		}
		fee = uint32(currentFee.Uint64())
	}
	// A plugin without SecurityModule has no status to read, and no halt this source knows about.
	return fee, resp.Result[1] && status != pluginStatusActive, nil
}

// fetchAllTicks finds the initialized ticks from the tick tree: each set bit of tickTreeRoot covers 256
// leaf words, whose tickBitmap words name the initialized ticks.
func (t *PoolTracker) fetchAllTicks(ctx context.Context, poolAddress string, root uint32,
	blockNumber *big.Int) ([]Tick, error) {
	var words []int16
	for r := root; r != 0; r &= r - 1 {
		first := bits.TrailingZeros32(r)<<8 - leafOffset
		for w := max(first, minLeafWord); w <= min(first+255, maxLeafWord); w++ {
			words = append(words, int16(w))
		}
	}

	var tickIndexes []int
	for _, chunk := range lo.Chunk(words, wordChunkSize) {
		bitmaps := make([]*big.Int, len(chunk))
		req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
		for i, word := range chunk {
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: poolAddress,
				Method: poolMethodTickBitmap,
				Params: []any{word},
			}, []any{&bitmaps[i]})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}

		for i, bitmap := range bitmaps {
			if bitmap == nil {
				continue
			}
			for bit := 0; bit < bitmap.BitLen(); bit++ {
				if bitmap.Bit(bit) == 1 {
					tickIndexes = append(tickIndexes, int(chunk[i])<<8+bit)
				}
			}
		}
	}

	return t.fetchTicks(ctx, poolAddress, tickIndexes, blockNumber)
}

// refreshTicks replaces the given tick indexes in ticks with their state at blockNumber, dropping the
// ones no longer initialized.
func (t *PoolTracker) refreshTicks(ctx context.Context, poolAddress string, ticks []Tick, indexes []int,
	blockNumber *big.Int) ([]Tick, error) {
	if len(indexes) == 0 {
		return ticks, nil
	}

	fresh, err := t.fetchTicks(ctx, poolAddress, indexes, blockNumber)
	if err != nil {
		return nil, err
	}

	byIndex := make(map[int]Tick, len(ticks)+len(fresh))
	for _, tick := range ticks {
		byIndex[tick.Index] = tick
	}
	for _, index := range indexes {
		delete(byIndex, index)
	}
	for _, tick := range fresh {
		byIndex[tick.Index] = tick
	}

	return sortedTicks(lo.Values(byIndex)), nil
}

// fetchTicks reads ticks(index) and returns the initialized ones, ascending.
func (t *PoolTracker) fetchTicks(ctx context.Context, poolAddress string, indexes []int,
	blockNumber *big.Int) ([]Tick, error) {
	ticks := make([]Tick, 0, len(indexes))
	for _, chunk := range lo.Chunk(indexes, tickChunkSize) {
		resps := make([]tickResp, len(chunk))
		req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
		for i, index := range chunk {
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: poolAddress,
				Method: poolMethodTicks,
				Params: []any{big.NewInt(int64(index))},
			}, []any{&resps[i]})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}

		for i, resp := range resps {
			if resp.LiquidityGross == nil || resp.LiquidityGross.Sign() == 0 {
				continue
			}
			var liquidityNet int256.Int
			if overflow := liquidityNet.SetFromBig(resp.LiquidityNet); overflow {
				return nil, ErrInt256Overflow
			}
			ticks = append(ticks, Tick{
				Index:          chunk[i],
				LiquidityGross: uint256.MustFromBig(resp.LiquidityGross),
				LiquidityNet:   &liquidityNet,
			})
		}
	}

	return sortedTicks(ticks), nil
}

func sortedTicks(ticks []Tick) []Tick {
	slices.SortFunc(ticks, func(a, b Tick) int { return a.Index - b.Index })
	return ticks
}

// touchedTicks returns the tickLower/tickUpper of every Mint and Burn the pool emitted in logs.
func touchedTicks(poolAddress string, logs []ethtypes.Log) []int {
	var indexes []int
	for _, log := range logs {
		if len(log.Topics) != 4 || !strings.EqualFold(log.Address.Hex(), poolAddress) {
			continue
		}
		if log.Topics[0] != mintEvent.ID && log.Topics[0] != burnEvent.ID {
			continue
		}
		indexes = append(indexes, int24FromTopic(log.Topics[2]), int24FromTopic(log.Topics[3]))
	}
	return lo.Uniq(indexes)
}

// int24FromTopic reads an indexed int24, which the topic holds sign-extended to 32 bytes.
func int24FromTopic(topic common.Hash) int {
	v := int(topic[29])<<16 | int(topic[30])<<8 | int(topic[31])
	if v&0x800000 != 0 {
		v -= 1 << 24
	}
	return v
}
