package utils

import (
	"cmp"

	mapset "github.com/deckarep/golang-set/v2"
)

func SetToSlice[T cmp.Ordered](set mapset.Set[T]) []T {
	if set == nil {
		return nil
	}
	return mapset.Sorted(set)
}
