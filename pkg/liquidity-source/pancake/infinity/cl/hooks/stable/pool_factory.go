package stable

import (
	"context"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	_ = cl.RegisterStableHookFactory(Factory)
	_ = cl.RegisterStableHookClassifier(ClassifyHooks)
)

func Factory(param *cl.HookParam) cl.Hook {
	h := &Hook{
		Hook:        cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLStable, param),
		exchange:    valueobject.ExchangePancakeInfinityCLStable,
		hookAddress: param.HookAddress,
	}

	if pool := param.Pool; pool != nil && len(param.HookExtra) > 0 {
		if inner, err := buildInner(pool, param.HookExtra); err == nil {
			h.inner = inner
		}
	}

	return h
}

// ClassifyHooks returns the subset of `hooks` registered as stable pools by
// any of `stableHookFactories`. For each unique non-zero hook it calls
// `stableHookFactory.get_implementation_address(hook)` per factory; a
// non-zero return means that factory owns the hook.
func ClassifyHooks(
	ctx context.Context,
	rpcClient *ethrpc.Client,
	stableHookFactories []string,
	hooks []common.Address,
) map[common.Address]bool {
	unique := lo.Uniq(lo.Filter(hooks, func(h common.Address, _ int) bool {
		return h != valueobject.AddrZero
	}))
	if len(unique) == 0 {
		return nil
	}

	results := make([][]common.Address, len(unique))
	req := rpcClient.NewRequest().SetContext(ctx)
	for hi, hook := range unique {
		results[hi] = make([]common.Address, len(stableHookFactories))
		for fi, factory := range stableHookFactories {
			req.AddCall(&ethrpc.Call{
				ABI:    hookFactoryABI,
				Target: factory,
				Method: "get_implementation_address",
				Params: []any{hook},
			}, []any{&results[hi][fi]})
		}
	}
	if _, err := req.TryAggregate(); err != nil {
		return nil
	}

	classified := make(map[common.Address]bool, len(unique))
	for hi, hook := range unique {
		if slices.ContainsFunc(results[hi], func(a common.Address) bool {
			return !valueobject.IsZeroAddress(a)
		}) {
			classified[hook] = true
		}
	}
	return classified
}
