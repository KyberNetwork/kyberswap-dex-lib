package usd0pp

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused    bool  `json:"paused"`
	EndTime   int64 `json:"endTime"`
	StartTime int64 `json:"startTime"`
}
