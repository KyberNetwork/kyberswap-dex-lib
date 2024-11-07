package stable

import "strings"

func NormalizePoolType(s string) string {
	if strings.EqualFold(s, poolTypeMetaStable) {
		return poolTypeStable
	}

	if strings.EqualFold(s, poolTypeStable) {
		return poolTypeStable
	}

	return s
}
