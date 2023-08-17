package entity

import "encoding/json"

const BestPathKey = "bestpaths"

type BestPath struct {
	// Pools list address pools that path swap through, length of pools = length of tokens - 1
	Pools []string
	// Tokens list tokens that path swap through
	Tokens []string
}

func (b BestPath) Encode() string {
	bytes, _ := json.Marshal(b)

	return string(bytes)
}

func DecodeBestPath(pathString string) *BestPath {
	var b BestPath
	err := json.Unmarshal([]byte(pathString), &b)

	if err != nil {
		return nil
	}

	return &b
}
