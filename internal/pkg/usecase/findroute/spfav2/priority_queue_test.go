package spfav2

import (
	"container/heap"
	"fmt"
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestPathPQ(t *testing.T) {
	paths := []*valueobject.Path{
		{
			Output: poolpkg.TokenAmount{
				AmountUsd: 100,
			},
		},
		{
			Output: poolpkg.TokenAmount{
				AmountUsd: 300,
			},
		},
		{
			Output: poolpkg.TokenAmount{
				AmountUsd: 200,
			},
		},
		{
			Output: poolpkg.TokenAmount{
				AmountUsd: 101,
			},
		},
		{
			Output: poolpkg.TokenAmount{
				AmountUsd: 302,
			},
		},
		{
			Output: poolpkg.TokenAmount{
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
