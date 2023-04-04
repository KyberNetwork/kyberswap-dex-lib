package dto

type IndexPoolsResult struct {
	// List of pool addresses that was failed to index
	FailedPoolAddresses []string
}

func NewIndexPoolsResult(failedPoolAddresses []string) *IndexPoolsResult {
	if len(failedPoolAddresses) == 0 {
		return nil
	}
	return &IndexPoolsResult{
		FailedPoolAddresses: failedPoolAddresses,
	}
}
