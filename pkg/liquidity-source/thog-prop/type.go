package thogprop

// StaticExtra is immutable per-pool metadata set once at discovery time.
type StaticExtra struct {
	PoolId string `json:"poolId"`
}

// Extra is rewritten end-to-end on each tracker refresh -- the ten packed
// state words plus the readiness/pause flags exactQuote() reads. Balances
// are indexed in the same order as Pool.Info.Tokens (tokenTable order).
type Extra struct {
	V1 string `json:"v1"`
	V2 string `json:"v2"`
	V3 string `json:"v3"`
	R1 string `json:"r1"`
	R2 string `json:"r2"`
	R3 string `json:"r3"`
	I1 string `json:"i1"`
	I2 string `json:"i2"`
	P1 string `json:"p1"`
	P2 string `json:"p2"`

	GloballyPaused bool `json:"globallyPaused"`
	RiskV3Ready    bool `json:"riskV3Ready"`
	PairRiskReady  bool `json:"pairRiskReady"`

	// SnapshotBlock is the block makerSnapshot() was pinned at -- the basis
	// for the age = executionBlock - postedBlock computation in math.go.
	SnapshotBlock uint64 `json:"snapshotBlock"`
}
