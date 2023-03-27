package entity

import "encoding/json"

const StatsEntity = "stats"

type PoolStatsItem struct {
	Size   int     `json:"poolSize"`
	Tvl    float64 `json:"tvl"`
	Tokens int     `json:"tokenSize"`
}

type Stats struct {
	Pools       map[string]PoolStatsItem
	TotalPools  int
	TotalTokens int
}

func (s Stats) Encode() (string, error) {
	bytes, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func DecodeStats(statsStr string) (s Stats, err error) {
	err = json.Unmarshal([]byte(statsStr), &s)

	return
}
