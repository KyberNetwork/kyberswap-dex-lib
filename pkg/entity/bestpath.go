package entity

import "github.com/bytedance/sonic"

const BestPathKey = "bestpaths"

type MinimalPath struct {
	// Pools list address pools that path swap through, length of pools = length of tokens - 1
	Pools []string
	// Tokens list tokens that path swap through
	Tokens []string
}

func (b MinimalPath) Encode() string {
	bytes, _ := sonic.Marshal(b)

	return string(bytes)
}

func DecodeBestPath(pathString string) *MinimalPath {
	var b MinimalPath
	err := sonic.Unmarshal([]byte(pathString), &b)

	if err != nil {
		return nil
	}

	return &b
}
