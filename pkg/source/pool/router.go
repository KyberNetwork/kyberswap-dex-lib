package pool

// RouterMetaInfo is the GetMetaInfo shape for pools whose executor calldata is
// just the router/venue address (push-payment: the executor transfers
// tokenIn to Router directly, then calls swap on it), as opposed to
// MetaInfo's ApprovalAddress which the executor approves instead.
type RouterMetaInfo struct {
	Router      string `json:"router"`
	BlockNumber uint64 `json:"blockNumber"`
}
