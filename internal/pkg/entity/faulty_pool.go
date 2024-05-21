package entity

type FaultyPoolTracker struct {
	Address string
	// number of times this pool appears in a build route request
	TotalCount int64
	// if estimate gas in build route failed, failed count = 1, otherwise is 0
	FailedCount int64
}
