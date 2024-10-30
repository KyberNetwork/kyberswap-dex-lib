package getrouteencode

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

func GetExcludedSources() []string {
	var excludedSources []string

	for source := range valueobject.RFQSourceSet {
		excludedSources = append(excludedSources, string(source))
	}

	return excludedSources
}
