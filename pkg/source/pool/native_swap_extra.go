package pool

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"

// NativeSwapExtraSource selects which of EncodingSwap's two extra blobs (aggregator-encoding's
// per-swap Extra, sourced from CalcAmountOutResult.SwapInfo, or PoolExtra, sourced from
// IPoolSimulator.GetMetaInfo) a DEX's registered native-swap value round-trips through.
type NativeSwapExtraSource int

const (
	NativeSwapExtraFromExtra NativeSwapExtraSource = iota
	NativeSwapExtraFromPoolExtra
)

type nativeSwapExtraEntry struct {
	source  NativeSwapExtraSource
	decoder func(raw any) (IPoolSupportNativeSwap, error)
}

// nativeSwapExtraRegistry lets a DEX package's Extra/PoolExtra value answer
// IPoolSupportNativeSwap on its own, keyed by pool type, for callers outside dex-lib (e.g.
// aggregator-encoding) that only have the JSON blob a swap already carries -- not a full,
// RPC-reconstructed IPoolSimulator.
var nativeSwapExtraRegistry = make(map[string]nativeSwapExtraEntry, 64)

// RegisterNativeSwapExtra registers, for poolType, how to decode the JSON-shaped value that
// ends up in EncodingSwap.Extra or EncodingSwap.PoolExtra (per source) into a T that already
// implements IPoolSupportNativeSwap. The [T IPoolSupportNativeSwap] constraint makes this
// registration itself a compile-time check that T implements the interface.
func RegisterNativeSwapExtra[T IPoolSupportNativeSwap](poolType string, source NativeSwapExtraSource) bool {
	if _, exists := nativeSwapExtraRegistry[poolType]; exists {
		panic(poolType + " native swap extra already registered")
	}
	nativeSwapExtraRegistry[poolType] = nativeSwapExtraEntry{
		source: source,
		decoder: func(raw any) (IPoolSupportNativeSwap, error) {
			v, err := util.AnyToStruct[T](raw)
			if err != nil {
				return nil, err
			}
			return *v, nil
		},
	}
	return true
}

// NativeSwapExtra looks up poolType's registered decoder and, if found, decodes the
// registered field (extra or poolExtra, whichever this poolType registered against) into a
// value implementing IPoolSupportNativeSwap. ok is false when poolType has no registration,
// or the registered field failed to decode -- callers should fall back to their own default
// in either case, not treat it as "does not support native swap".
func NativeSwapExtra(poolType string, extra, poolExtra any) (result IPoolSupportNativeSwap, ok bool) {
	entry, registered := nativeSwapExtraRegistry[poolType]
	if !registered {
		return nil, false
	}

	raw := extra
	if entry.source == NativeSwapExtraFromPoolExtra {
		raw = poolExtra
	}

	v, err := entry.decoder(raw)
	if err != nil {
		return nil, false
	}
	return v, true
}
