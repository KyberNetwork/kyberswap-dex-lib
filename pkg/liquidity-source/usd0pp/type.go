package usd0pp

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused    bool  `json:"paused"`
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
}

type Gas struct {
	Mint int64
}
