package common

func Filter[T any](items []T, filters ...ItemFilter[T]) []T {
	filteredObjects := make([]T, 0, len(items))

	for _, pool := range items {
		valid := true

		for _, filter := range filters {
			if !filter(pool) {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		filteredObjects = append(filteredObjects, pool)
	}

	return filteredObjects
}

type ItemFilter[T any] func(item T) bool
