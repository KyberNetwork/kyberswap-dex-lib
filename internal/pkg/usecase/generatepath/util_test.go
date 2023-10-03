package generatepath

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/stretchr/testify/require"
)

func TestHashBestPaths(t *testing.T) {
	p1 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa"},
		Tokens: []string{"0xcccc", "0xdddd"},
	}
	p2 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeee"},
	}
	p3 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeef"},
	}
	p4 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeee"},
	}
	require.NotEqual(t, hashBestPath(p1), hashBestPath(p2))
	require.NotEqual(t, hashBestPath(p2), hashBestPath(p3))
	require.Equal(t, hashBestPath(p2), hashBestPath(p4))
}

func TestDedupBestPaths(t *testing.T) {
	p1 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa"},
		Tokens: []string{"0xcccc", "0xdddd"},
	}
	p2 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeee"},
	}
	p3 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeef"},
	}
	p4 := &entity.MinimalPath{
		Pools:  []string{"0xaaaa", "0xbbbb"},
		Tokens: []string{"0xcccc", "0xdddd", "0xeeee"},
	}
	deduped := dedupBestPaths([]*entity.MinimalPath{p1, p2, p3, p4})
	expected := []*entity.MinimalPath{p1, p2, p3}
	require.EqualValues(t, deduped, expected)
}
