package dto

type IndexPoolsResult struct {
	// List of pool addresses that was failed to index
	FailedPoolAddresses []string

	// Number of old pool skipped
	OldPoolCount int
}

func NewIndexPoolsResult(failedPoolAddresses []string, oldPoolCount int) *IndexPoolsResult {
	return &IndexPoolsResult{
		FailedPoolAddresses: failedPoolAddresses,
		OldPoolCount:        oldPoolCount,
	}
}
