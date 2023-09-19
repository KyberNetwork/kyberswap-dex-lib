package uniswapv3pt

import (
	"bytes"
	"math/big"
	"text/template"
)

type PoolsListQueryParams struct {
	AllowSubgraphError     bool
	LastCreatedAtTimestamp *big.Int
	First                  int
	Skip                   int
}

func getPoolsListQuery(allowSubgraphError bool, lastCreatedAtTimestamp *big.Int, first, skip int) string {
	var tpl bytes.Buffer
	td := PoolsListQueryParams{
		allowSubgraphError,
		lastCreatedAtTimestamp,
		first,
		skip,
	}

	// Add subgraphError: allow
	t, err := template.New("poolsListQuery").Parse(`{
		pools(
			{{ if .AllowSubgraphError }}subgraphError: allow,{{ end }}
			where: {
				createdAtTimestamp_gte: {{ .LastCreatedAtTimestamp }}
			},
			first: {{ .First }},
			skip: {{ .Skip }},
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			feeTier
			token0 {
				id
				name
				symbol
				decimals
			}
			token1 {
				id
				name
				symbol
				decimals
			}
		}
	}`)

	if err != nil {
		panic(err)
	}

	err = t.Execute(&tpl, td)

	if err != nil {
		panic(err)
	}

	return tpl.String()
}
