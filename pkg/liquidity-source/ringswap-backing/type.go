package ringswapbacking

import "math/big"

type BackingSourceState struct {
	Reserve0        *big.Int
	Reserve1        *big.Int
	WrapperBuffer0  *big.Int
	WrapperBuffer1  *big.Int
	RecallCapacity0 *big.Int
	RecallCapacity1 *big.Int
}

// go-ethereum wraps a function's single tuple output before copying its components.
type backingSourceStateResult struct {
	State BackingSourceState
}

type RouteQuote struct {
	AmountOut      *big.Int
	WrapperBuffer  *big.Int
	RecallCapacity *big.Int
	RecallRequired bool
	Executable     bool
}

type routeQuoteResult struct {
	Quote RouteQuote
}

type Extra struct {
	WrapperBuffer0  *big.Int `json:"wrapperBuffer0"`
	WrapperBuffer1  *big.Int `json:"wrapperBuffer1"`
	RecallCapacity0 *big.Int `json:"recallCapacity0"`
	RecallCapacity1 *big.Int `json:"recallCapacity1"`
}

type StaticExtra struct {
	RouterAddress       string `json:"routerAddress"`
	PairAddress         string `json:"pairAddress"`
	Wrapper0            string `json:"wrapper0"`
	Wrapper1            string `json:"wrapper1"`
	ReplaceOrdinaryPair bool   `json:"replaceOrdinaryPair"`
	NoRecallGasToken0   int64  `json:"noRecallGasToken0"`
	NoRecallGasToken1   int64  `json:"noRecallGasToken1"`
	RecallGasToken0     int64  `json:"recallGasToken0"`
	RecallGasToken1     int64  `json:"recallGasToken1"`
}

type SwapInfo struct {
	RouterAddress  string `json:"routerAddress"`
	UnderlyingPair string `json:"underlyingPair"`
	WrapperIn      string `json:"wrapperIn"`
	WrapperOut     string `json:"wrapperOut"`
	UseRecall      bool   `json:"useRecall"`
}

type PoolMeta struct {
	ApprovalAddress      string `json:"approvalAddress"`
	RouterAddress        string `json:"routerAddress"`
	UnderlyingPair       string `json:"underlyingPair"`
	SingleUse            bool   `json:"singleUse"`
	ReplacesOrdinaryPair bool   `json:"replacesOrdinaryPair"`
}

type PoolsListUpdaterMetadata struct {
	KnownRouters []string `json:"knownRouters"`
	KnownPairs   []string `json:"knownPairs"`
}
