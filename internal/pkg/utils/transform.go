package utils

import (
	"strings"
)

const SliceParamsItemSeparator = ","

func TransformSliceParams(params string) []string {
	var items []string

	if len(params) == 0 {
		return items
	}

	for _, item := range strings.Split(params, SliceParamsItemSeparator) {
		items = append(
			items,
			CleanUpParam(item),
		)
	}

	return items
}

func CleanUpParam(param string) string {
	return strings.ToLower(strings.TrimSpace(param))
}
