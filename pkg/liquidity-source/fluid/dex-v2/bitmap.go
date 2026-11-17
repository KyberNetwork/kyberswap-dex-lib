package dexv2

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
)

// fetchPoolTicksFromBitmap enumerates initialized ticks at the same block as the scalar state.
func (t *PoolTracker) fetchPoolTicksFromBitmap(ctx context.Context, p entity.Pool, block uint64) ([]Tick, error) {
	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		return nil, err
	}
	if static.TickSpacing == 0 || block == 0 {
		return nil, fmt.Errorf("missing tick spacing or snapshot block")
	}
	spacing := int(static.TickSpacing)
	// Arithmetic shifts implement floor division for negative compressed ticks.
	minCompressed := MIN_TICK / spacing
	if MIN_TICK%spacing != 0 {
		minCompressed--
	}
	minWord, maxWord := minCompressed>>8, (MAX_TICK/spacing)>>8
	dexID, dexType := parseFluidDexV2PoolAddress(p.Address)
	var indexes []int
	for first := minWord; first <= maxWord; first += 256 {
		end := min(first+256, maxWord+1)
		words := make([]*big.Int, end-first)
		req := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(block))
		for word := first; word < end; word++ {
			slot := calculateTripleMappingStorageSlot(2, dexType, common.HexToHash(dexID), word)
			req.AddCall(&ethrpc.Call{ABI: abis.DexV2ABI, Target: t.config.Dex, Method: "readFromStorage", Params: []any{slot}}, []any{&words[word-first]})
		}
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}
		for i, bits := range words {
			if bits == nil {
				return nil, fmt.Errorf("missing tick bitmap word")
			}
			for bit := 0; bit < 256; bit++ {
				if bits.Bit(bit) != 0 {
					indexes = append(indexes, ((first+i)*256+bit)*spacing)
				}
			}
		}
	}
	raw, err := t.queryTicksFromRPC(ctx, p.Address, indexes, block)
	if err != nil {
		return nil, err
	}
	ticks := make([]Tick, 0, len(raw))
	for _, tick := range raw {
		if tick.LiquidityGross.Sign() > 0 {
			ticks = append(ticks, Tick{Index: tick.TickIdx, LiquidityGross: tick.LiquidityGross, LiquidityNet: tick.LiquidityNet})
		}
	}
	return ticks, nil
}
