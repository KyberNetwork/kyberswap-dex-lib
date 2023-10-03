package generatepath

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func hashBestPath(path *entity.MinimalPath) uint64 {
	buf := make([]byte, 8)
	h := fnv.New64()

	binary.BigEndian.PutUint64(buf, uint64(len(path.Pools)))
	h.Write(buf)
	for _, p := range path.Pools {
		h.Write([]byte(p))
	}

	binary.BigEndian.PutUint64(buf, uint64(len(path.Tokens)))
	h.Write(buf)
	for _, t := range path.Tokens {
		h.Write([]byte(t))
	}

	return h.Sum64()
}

func dedupBestPaths(paths []*entity.MinimalPath) []*entity.MinimalPath {
	s := make(map[uint64]struct{})
	var deduped []*entity.MinimalPath
	for _, p := range paths {
		h := hashBestPath(p)
		if _, ok := s[h]; !ok {
			s[h] = struct{}{}
			deduped = append(deduped, p)
		}
	}
	return deduped
}
