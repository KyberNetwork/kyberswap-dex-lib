package shared

import "slices"

// PoolTokens returns what a pool with base pools reports as its tokens: its own tokens first, in their
// order, which is the order of its reserves, so the i-th token pairs with the i-th reserve; then the
// tokens its base pools add, once each and sorted. The result depends on neither map's iteration order.
// Without base pools it is the pool's own tokens.
func PoolTokens(tokens []string, basePools map[string]IBasePool) []string {
	if len(basePools) == 0 {
		return tokens
	}

	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		seen[token] = struct{}{}
	}

	var extra []string
	for _, basePool := range basePools {
		for _, token := range basePool.GetTokens() {
			if _, ok := seen[token]; !ok {
				seen[token] = struct{}{}
				extra = append(extra, token)
			}
		}
	}
	slices.Sort(extra)

	result := make([]string, 0, len(tokens)+len(extra))
	result = append(result, tokens...)
	return append(result, extra...)
}
