package api

import (
	"strings"
)

const SliceParamsItemSeparator = ","

func transformSliceParams(params string) []string {
	var items []string

	if len(params) == 0 {
		return items
	}

	for _, item := range strings.Split(params, SliceParamsItemSeparator) {
		items = append(
			items,
			cleanUpParam(item),
		)
	}

	return items
}

func cleanUpParam(param string) string {
	return strings.ToLower(strings.TrimSpace(param))
}
