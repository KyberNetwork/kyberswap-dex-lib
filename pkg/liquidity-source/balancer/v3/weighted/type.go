package weighted

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
)

type Extra struct {
	*shared.Extra
	NormalizedWeights []*uint256.Int `json:"normalizedWeights"`
	MinTokenBalances  []*uint256.Int `json:"minTokenBalances,omitempty"`
}

type RpcResult struct {
	shared.RpcResult
	NormalizedWeights []*big.Int
	MinTokenBalances  []*big.Int
	Version           string
}

// ResolveMinTokenBalances resolves immutable WeightedPool v2 minTokenBalances from the RPC
// result, previously persisted Extra, and version() JSON ({"name":"WeightedPool","version":N,...}).
// A v2 pool never silently falls back to v1 math if an RPC sub-call fails transiently.
func ResolveMinTokenBalances(prevExtra, rpcVersion string, rpcMinBals []*big.Int) ([]*uint256.Int, error) {
	if len(rpcMinBals) > 0 {
		return shared.FromBigs(rpcMinBals), nil
	}
	var prev Extra
	if prevExtra != "" && json.Unmarshal([]byte(prevExtra), &prev) == nil && len(prev.MinTokenBalances) > 0 {
		return prev.MinTokenBalances, nil
	}
	if rpcVersion == "" {
		return nil, ErrMissingPoolVersion
	}
	var meta struct {
		Version int `json:"version"`
	}
	if json.Unmarshal([]byte(rpcVersion), &meta) == nil && meta.Version >= 2 {
		return nil, ErrMissingMinTokenBalances
	}
	return nil, nil
}
