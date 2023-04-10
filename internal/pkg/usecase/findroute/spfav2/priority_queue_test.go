package spfav2

import (
	"container/heap"
	"fmt"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
)

func TestPathPQ(t *testing.T) {
	paths := []*core.Path{
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 100,
			},
		},
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 300,
			},
		},
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 200,
			},
		},
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 101,
			},
		},
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 302,
			},
		},
		{
			Output: poolPkg.TokenAmount{
				AmountUsd: 200,
			},
		},
	}
	cmpFunc := func(a, b int) bool {
		return paths[a].CompareTo(paths[b], true) < 0
	}
	pq := NewPriorityQueue(len(paths), cmpFunc)
	for pq.Len() > 0 {
		path1 := paths[pq.Top().(int)]
		path := paths[heap.Pop(pq).(int)]
		fmt.Println(path1.Output.AmountUsd, path.Output.AmountUsd)
	}
}
