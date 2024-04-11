package dto

type IndexPoolsCommand struct {
	PoolAddresses []string

	// pools with timestamp older than this will be ignored (default to 0 to index all pools)
	IgnorePoolsBeforeTimestamp int64
}
