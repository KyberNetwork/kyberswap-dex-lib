//go:build livegate

package twocryptong

// SetNowUnix pins the simulator's block.timestamp for the live comparison harness.
func SetNowUnix(ts int64) { nowUnix = func() int64 { return ts } }
