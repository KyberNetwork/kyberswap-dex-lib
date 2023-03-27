package promm

import "time"

const (
	defaultGraphQLRequestTimeout = 20 * time.Second

	graphSkipLimit  = 5000
	graphFirstLimit = 1000
)
