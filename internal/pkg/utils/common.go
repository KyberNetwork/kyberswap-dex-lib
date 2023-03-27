package utils

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/util/env"
)

const (
	EmptyString       = ""
	Zero        int64 = 0
)

func IsEnable(flag string) bool {
	return flag == "1" || flag == "true"
}
func Chunks(xs []string, chunkSize int) [][]string {
	if len(xs) == 0 {
		return nil
	}
	divided := make([][]string, (len(xs)+chunkSize-1)/chunkSize)
	prev := 0
	i := 0
	till := len(xs) - chunkSize
	for prev < till {
		next := prev + chunkSize
		divided[i] = xs[prev:next]
		prev = next
		i++
	}
	divided[i] = xs[prev:]
	return divided
}

func IsAllowSubgraphError() bool {
	return env.ParseBoolFromEnv(envvar.AllowSubgraphError, false)
}
