package getrouteencode

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

func GetExcludedSources() []string {
	var excludedSources []string

	for source := range valueobject.RFQSourceSet {
		excludedSources = append(excludedSources, string(source))
	}

	return excludedSources
}

func GetSourcesAfterExclude(availableSources []string) []string {
	var result []string
	for _, dex := range availableSources {
		if _, exist := valueobject.RFQSourceSet[valueobject.Exchange(dex)]; !exist {
			result = append(result, dex)
		}
	}
	return result
}
