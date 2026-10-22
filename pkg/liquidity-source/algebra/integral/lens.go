package integral

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abis/lens.json
var lensJSON []byte

// lens is AlgebraIntegralLens, released by KyberNetwork/ephemeral-contracts.
var lens = ephemeral.MustLoad("AlgebraIntegralLens", lensJSON)

// snapshot is what getSnapshot returns, one field per output in the ABI's order.
type snapshot struct {
	TickSpacing  *big.Int
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Liquidity    *big.Int
	BlockNumber  *big.Int
	NextTick     *big.Int
	Done         bool
	Ticks        []struct {
		Tick           *big.Int
		LiquidityNet   *big.Int
		LiquidityGross *big.Int
	}
}

// lensStartTick is TickMath.MIN_TICK, the sentinel the pool's tick list starts from.
const lensStartTick = -887272

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block        uint64
	sqrtPriceX96 *big.Int
	tick         int
	liquidity    *big.Int
	ticks        []tickspkg.Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens, paging along the tick
// list when it does not fit one call, and checks the ticks against the pool's liquidity.
func (d *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	pool := common.HexToAddress(address)
	var s lensSnapshot

	pin, err := ephemeral.Pin(ctx, d.EthrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	read, err := ephemeral.Scan(ctx, lensStartTick, pin,
		func(ctx context.Context, start int, block *big.Int) (ephemeral.Page[int], error) {
			var out snapshot
			err := ephemeral.Call(ctx, d.EthrpcClient.GetETHClient(), lens, &out, d.config.LensGas, block,
				pool, big.NewInt(int64(start)))
			if err != nil {
				return ephemeral.Page[int]{}, err
			}
			if s.liquidity == nil { // the first page
				s.sqrtPriceX96 = out.SqrtPriceX96
				s.tick = int(out.Tick.Int64())
				s.liquidity = out.Liquidity
			}
			for _, pt := range out.Ticks {
				s.ticks = append(s.ticks, tickspkg.Tick{
					TickIdx:        int(pt.Tick.Int64()),
					LiquidityGross: pt.LiquidityGross,
					LiquidityNet:   pt.LiquidityNet,
				})
			}
			return ephemeral.Page[int]{Block: out.BlockNumber.Uint64(), Next: int(out.NextTick.Int64()),
				Done: out.Done}, nil
		})
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	s.block = read

	checked := make([]ephemeral.Tick, len(s.ticks))
	for i, tk := range s.ticks {
		checked[i] = ephemeral.Tick{Index: tk.TickIdx, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	if err := ephemeral.CheckTicks(checked, s.tick, s.liquidity); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, s.block, err)
	}

	return &s, nil
}

// hasState reports whether a tracker has given the pool its state. Only a tracker writes its
// liquidity, 0 included; the pool factory writes no Extra at all.
func hasState(p entity.Pool) bool {
	var extra struct {
		Liquidity json.RawMessage `json:"liq"`
	}
	return len(p.Extra) > 0 && json.Unmarshal([]byte(p.Extra), &extra) == nil &&
		len(extra.Liquidity) > 0 && string(extra.Liquidity) != "null"
}

// withoutState drops the state of p its logs built, its liquidity and ticks, keeping the rest of
// the Extra, the plugin's timepoints and fee config among it, and records that a lens snapshot must
// reach due: p is then a pool without state, which the next update reads whole.
func withoutState(p entity.Pool, due uint64) entity.Pool {
	extra := map[string]json.RawMessage{}
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	delete(extra, "liq")
	delete(extra, "ticks")
	extra["lensDue"] = json.RawMessage(strconv.FormatUint(max(tickspkg.LensDue(p), due), 10))
	if b, err := json.Marshal(extra); err == nil {
		p.Extra = string(b)
	}
	return p
}
