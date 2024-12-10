package dto

type IndexPoolsResult struct {
	// Total number of pools should be indexed
	TotalCount int

	// List of pool addresses that was failed to index
	FailedPoolAddresses []string

	// Number of old pool skipped
	OldPoolCount int
}

func NewIndexPoolsResult(totalCount int, failedPoolAddresses []string, oldPoolCount int) *IndexPoolsResult {
	return &IndexPoolsResult{
		TotalCount:          totalCount,
		FailedPoolAddresses: failedPoolAddresses,
		OldPoolCount:        oldPoolCount,
	}
}
