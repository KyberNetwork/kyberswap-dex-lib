package getrouteencode

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

func GetExcludedSources() []string {
	var excludedSources []string

	for source := range valueobject.RFQSourceSet {
		excludedSources = append(excludedSources, string(source))
	}

	return excludedSources
}
