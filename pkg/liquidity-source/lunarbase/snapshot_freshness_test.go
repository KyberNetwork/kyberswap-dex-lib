package lunarbase

import "testing"

func TestSnapshotFreshnessBoundaryAndKnownZero(t *testing.T) {
	for _, hash := range []string{"", "0xverified"} {
		e := Extra{BlockHash: hash, LatestUpdateBlock: 100, BlockDelay: 25}
		for _, b := range []uint64{99, 100, 124, 125, 126} {
			if e.IsStale(b) != (b >= 125) {
				t.Fatalf("hash=%s block=%d", hash, b)
			}
		}
	}
	known := Extra{BlockHash: "0xverified", LatestUpdateBlock: 0, BlockDelay: 25}
	if known.IsStale(24) || !known.IsStale(25) || !known.IsStale(900) {
		t.Fatal("a verified zero update block must expire at its real boundary")
	}
	legacy := Extra{BlockDelay: 25}
	if legacy.IsStale(900) {
		t.Fatal("legacy missing metadata compatibility changed")
	}
	// A zero delay is rejected by the RPC reader. If such verified metadata
	// is restored from storage, do not interpret it as an unlimited lifetime.
	known.BlockDelay = 0
	if !known.IsStale(0) || !known.IsStale(900) {
		t.Fatal("known zero delay accepted")
	}
}
